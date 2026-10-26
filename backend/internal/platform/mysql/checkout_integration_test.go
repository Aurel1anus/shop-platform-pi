package mysqlstore_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/cart"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/catalog"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/order"
	mysqlstore "github.com/LYP-leo/xzxg-shop/commerce/internal/platform/mysql"
)

func TestCheckoutIntegration(t *testing.T) {
	dsn := os.Getenv("COMMERCE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set COMMERCE_TEST_MYSQL_DSN to a dedicated disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := mysqlstore.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := repo.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := fmt.Sprintf("integration_%d", time.Now().UnixNano())
	account, err := repo.CreateAccount(ctx, domain.AccountCreateInput{Username: id, PasswordHash: "unused-test-hash", DisplayName: id, Role: domain.AccountRoleUser})
	if err != nil {
		t.Fatal(err)
	}
	product, err := catalog.New(repo).CreateProduct(ctx, domain.ProductUpsertInput{ProductID: id, MerchantID: "m_001", Name: "Transaction test", Price: "7.50", StockQuantity: 20})
	if err != nil {
		t.Fatal(err)
	}
	skus := repo.ListProductSKUs(ctx, product.ProductID)
	if len(skus) != 1 {
		t.Fatalf("SKUs: %+v", skus)
	}
	cartService, orderService := cart.New(repo), order.New(repo)
	if _, ok := cartService.AddCartItem(ctx, account.AccountID, product.ProductID, skus[0].SkuID, 2); !ok {
		t.Fatal("add cart failed")
	}
	// Reserve the SKU first, then fail the product reservation. Both changes must roll back.
	if _, err := db.ExecContext(ctx, "UPDATE products SET stock_quantity = 0 WHERE product_id = ?", product.ProductID); err != nil {
		t.Fatal(err)
	}
	if _, ok := orderService.CreateOrderFromCart(ctx, account.AccountID); ok {
		t.Fatal("checkout should fail with insufficient product stock")
	}
	var stock int
	if err := db.QueryRowContext(ctx, "SELECT stock_quantity FROM product_skus WHERE sku_id = ?", skus[0].SkuID).Scan(&stock); err != nil || stock != 20 {
		t.Fatalf("SKU reservation was not rolled back: stock=%d err=%v", stock, err)
	}
	if len(repo.GetCart(ctx, account.AccountID).Items) != 1 || len(repo.ListUserOrders(ctx, account.AccountID)) != 0 {
		t.Fatal("failed checkout modified cart or created an order")
	}
	if _, err := db.ExecContext(ctx, "UPDATE products SET stock_quantity = 20 WHERE product_id = ?", product.ProductID); err != nil {
		t.Fatal(err)
	}
	orders, ok := orderService.CreateOrderFromCart(ctx, account.AccountID)
	if !ok || len(orders) != 1 || orders[0].PayAmount != "15.00" {
		t.Fatalf("checkout: ok=%v orders=%+v", ok, orders)
	}
	if len(repo.GetCart(ctx, account.AccountID).Items) != 0 {
		t.Fatal("successful checkout did not clear selected items")
	}
	orderID := orders[0].OrderID
	if _, ok := orderService.GetOrder(ctx, "another-account", orderID); ok {
		t.Fatal("order leaked to another account")
	}
	paid, _, ok := orderService.PayOrder(ctx, account.AccountID, orderID, "mock")
	if !ok || paid.Status != "pending_ship" {
		t.Fatalf("pay: %+v", paid)
	}
	if _, ok := orderService.UpdateOrderStatus(ctx, "m_001", orderID, "shipped"); !ok {
		t.Fatal("ship failed")
	}
	completed, ok := orderService.ConfirmReceipt(ctx, account.AccountID, orderID)
	if !ok || completed.Status != "completed" {
		t.Fatalf("receipt: %+v", completed)
	}
}
