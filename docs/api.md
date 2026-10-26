# 业务 API 清单

基础地址：`http://localhost:8081/api/v1`。路径与原业务接口保持一致。

除公开商品浏览、健康检查和登录注册外，按接口校验用户/商家/管理员角色。受保护请求携带 `Authorization: Bearer <token>`。用户操作的 account_id 来自凭证，不从请求体取值。

## 身份与角色

- 用户：购物车、订单、优惠券、语音。
- 商家：自身商品、订单履约、促销、评价回复、知识资料。
- 管理员：账号、商家、商品、订单、促销、评价、配置、知识资料。
- 文件：上传需登录；下载限文件拥有者或管理员。

## 注册路由

| 方法与路径 | Handler |
|---|---|
| `GET /api/v1/health` | `handleHealth` |
| `POST /api/v1/auth/login` | `handleLogin` |
| `POST /api/v1/auth/register` | `handleRegister` |
| `POST /api/v1/auth/logout` | `handleLogout` |
| `GET /api/v1/auth/me` | `handleMe` |
| `GET /api/v1/account/profile` | `handleAccountProfile` |
| `PATCH /api/v1/account/profile` | `handleAccountProfile` |
| `PATCH /api/v1/account/contact` | `handleAccountContact` |
| `DELETE /api/v1/account` | `handleDeleteAccount` |
| `POST /api/v1/uploads/avatar` | `handleUploadAvatar` |
| `GET /api/v1/categories/tree` | `handleListCategories` |
| `GET /api/v1/merchants` | `handleListMerchants` |
| `GET /api/v1/products` | `handleListProducts` |
| `GET /api/v1/products/` | `handleProductAction` |
| `GET /api/v1/promotions` | `handleListPromotions` |
| `GET /api/v1/coupons/available` | `handleListCoupons` |
| `GET /api/v1/coupons/mine` | `handleListUserCoupons` |
| `POST /api/v1/coupons/` | `handleCouponAction` |
| `POST /api/v1/files` | `handleUploadFile` |
| `GET /api/v1/files/` | `handleGetFile` |
| `GET /api/v1/merchant/documents` | `handleListMerchantDocuments` |
| `POST /api/v1/merchant/documents` | `handleCreateMerchantDocument` |
| `POST /api/v1/merchant/unstructured-ingestions` | `handleCreateMerchantUnstructuredIngestion` |
| `GET /api/v1/merchant/orders` | `handleListMerchantOrders` |
| `PATCH /api/v1/merchant/orders/` | `handleUpdateMerchantOrder` |
| `GET /api/v1/merchant/promotions` | `handleListMerchantPromotions` |
| `POST /api/v1/merchant/promotions` | `handleCreateMerchantPromotion` |
| `PATCH /api/v1/merchant/promotions/` | `handleUpdateMerchantPromotion` |
| `GET /api/v1/merchant/reviews` | `handleListMerchantReviews` |
| `POST /api/v1/merchant/reviews/` | `handleMerchantReviewAction` |
| `POST /api/v1/merchant/products` | `handleCreateMerchantProduct` |
| `PATCH /api/v1/merchant/products/` | `handleUpdateMerchantProduct` |
| `DELETE /api/v1/merchant/products/` | `handleDeleteMerchantProduct` |
| `GET /api/v1/admin/accounts` | `handleListAdminAccounts` |
| `PATCH /api/v1/admin/accounts/` | `handleUpdateAdminAccount` |
| `GET /api/v1/admin/configs` | `handleListAdminConfigs` |
| `PATCH /api/v1/admin/configs/` | `handleUpdateAdminConfig` |
| `GET /api/v1/admin/documents` | `handleListAdminDocuments` |
| `POST /api/v1/admin/unstructured-ingestions` | `handleCreateAdminUnstructuredIngestion` |
| `GET /api/v1/admin/orders` | `handleListAdminOrders` |
| `PATCH /api/v1/admin/orders/` | `handleUpdateAdminOrder` |
| `GET /api/v1/admin/promotions` | `handleListAdminPromotions` |
| `POST /api/v1/admin/promotions` | `handleCreateAdminPromotion` |
| `PATCH /api/v1/admin/promotions/` | `handleUpdateAdminPromotion` |
| `GET /api/v1/admin/reviews` | `handleListAdminReviews` |
| `PATCH /api/v1/admin/reviews/` | `handleUpdateAdminReview` |
| `GET /api/v1/admin/merchants` | `handleListAdminMerchants` |
| `GET /api/v1/admin/products` | `handleListAdminProducts` |
| `PATCH /api/v1/admin/products/` | `handleUpdateAdminProduct` |
| `PATCH /api/v1/admin/merchants/` | `handleUpdateAdminMerchant` |
| `GET /api/v1/cart` | `handleGetCart` |
| `GET /api/v1/cart/discount-preview` | `handleCartDiscountPreview` |
| `POST /api/v1/cart/items` | `handleAddCartItem` |
| `PATCH /api/v1/cart/items/` | `handleCartItemAction` |
| `DELETE /api/v1/cart/items/` | `handleCartItemAction` |
| `GET /api/v1/orders` | `handleListUserOrders` |
| `POST /api/v1/orders:checkout` | `handleCheckout` |
| `GET /api/v1/orders/` | `handleUserOrderAction` |
| `POST /api/v1/orders/` | `handleUserOrderAction` |

## 动作路径

- 商品：`GET /products/{product_id}`、`GET /products/{product_id}/reviews`。
- 订单：`GET /orders/{order_id}`、`POST /orders/{order_id}:pay`、`:cancel`、`:confirm-receipt`。
- 订单评价：`POST /orders/{order_id}/items/{order_item_id}:review`。
- 优惠券领取：`POST /coupons/{coupon_id}:claim`。
- 商家评价回复：`POST /merchant/reviews/{review_id}:reply`。

最终路径解析以 internal/httpapi/common.go 为准。

## 常用请求

登录：`POST /auth/login`，JSON：`{"username":"user","password":"user123456"}`，返回 token 与 account。

加购：`POST /cart/items`，JSON：`{"product_id":"p_001","sku_id":"","quantity":1}`。省略 SKU 时使用默认候选；指定 SKU 必须属于该商品。返回购物车。

结算：`POST /orders:checkout`，结算当前用户购物车选中项，按商家拆单，返回 `items` 订单数组。

商品、订单和管理列表支持 `page`、`page_size`。商品列表支持 `keyword`、`category_id`。数据字段沿用原来的 camelCase/snake_case 约定，尚未进行破坏性统一。

## 原始知识资料

`GET /merchant/documents`、`GET /admin/documents` 返回原始 content、metadata、source_url、content_hash。上传与非结构化入库不触发切块，状态为 pending_index。

## 暂未接入的接口

`/agent/*`、`/eval/*`、`/admin/agent/*`、`/admin/prompts*`、`/admin/evals*`、`/admin/vector/*` 与图片语义搜索返回 501。Agent 服务接入方案见 agent-boundary.md。
