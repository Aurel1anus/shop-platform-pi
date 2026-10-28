---
name: shopping
description: 按关键词分页浏览真实商品、根据商品 ID 查询详情及各 SKU 规格、价格与库存；语义检索推荐尚未接入。
---

1. 用户提供商品 ID 时，调用 get_product 查询详情。只有名称或需求时，先调用 list_products 查询关键词；需要详情时使用实际列表返回的商品 ID，不猜测 ID。空关键词可以浏览全部商品。
2. list_products 支持 keyword、page、page_size（默认页码 1、每页 10 条）；total 是匹配总数，items 是当前页，不要混淆。需要下一页时保留关键词并增加页码；空列表如实说明本页无结果，total 为 0 才表示无匹配商品。多词可能匹配任意词，不把关键词查询说成语义推荐或保证满足全部条件。
3. 根据实际结果说明名称、价格、适用情况和风险；没有返回的字段明确说明未知。
4. 查询颜色、版本、规格价格或库存数量时，用已知 product_id 调用 list_product_skus；返回 skuId、skuName、specs、price、stockQuantity、stockStatus。不要把列表或详情中的单个 SKU 当成全部规格，也不要把 in_stock 当成库存数量。
5. SKU 查询支持 page、page_size，默认第一页、每页 10 条；需要全部规格时继续翻页，不能只看第一页就宣称已列完。空页且 total>0 表示本页无数据；total=0 只能说明未查到规格，现有后端不能据此区分无规格、商品不存在或部分数据库错误；需要确认商品是否存在时调用 get_product。
6. 工具失败时说明错误，不声称查询成功。不加购、不下单、不支付；这些工具尚未开放。

shop_status 只报告接入状态，不探测后端，不能作为商品查询依据。
