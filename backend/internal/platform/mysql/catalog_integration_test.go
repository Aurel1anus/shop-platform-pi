package mysqlstore_test

import (
	"context"
	"os"
	"testing"
	"time"

	mysqlstore "github.com/LYP-leo/xzxg-shop/commerce/internal/platform/mysql"
)

func TestProductPaginationIntegration(t *testing.T) {
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
	for _, tc := range []struct{ name, keyword, category string }{
		{"all", "", ""},
		{"keyword", "Camera", ""},
		{"category", "", "c_phone"},
		{"no_matches", "no_such_product_7e3b09", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			all := repo.ListProducts(ctx, tc.keyword, tc.category)
			if tc.name != "no_matches" && len(all) == 0 {
				t.Fatal("expected seeded matching products")
			}
			for _, size := range []int{1, 2} {
				items, total := repo.ListProductsPage(ctx, tc.keyword, tc.category, 1, size)
				want := min(size, len(all))
				if total != len(all) || len(items) != want {
					t.Fatalf("size=%d: total=%d items=%d; want total=%d items=%d", size, total, len(items), len(all), want)
				}
			}
			items, total := repo.ListProductsPage(ctx, tc.keyword, tc.category, len(all)+1, 1)
			if len(items) != 0 || total != len(all) {
				t.Fatalf("past last page: total=%d items=%d; want total=%d items=0", total, len(items), len(all))
			}
		})
	}
}
