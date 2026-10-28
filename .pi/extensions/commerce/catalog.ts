import type { ExtensionAPI } from '@earendil-works/pi-coding-agent';
import { Type } from 'typebox';
import { getProduct, listProducts, listProductSkus } from '../../../src/clients/commerce-python.ts';

export function registerCatalogTools(pi: ExtensionAPI) {
  pi.registerTool({
    name: 'list_product_skus',
    label: '查询商品规格',
    description: '根据已知商品 ID 分页查询真实 SKU：规格名称、规格字典、各规格价格与库存。items 只包含当前页，total 为规格总数，需要全部规格时继续翻页。空结果只能说明未查到规格，不能判断商品不存在；可用 get_product 确认商品。不执行写操作。',
    parameters: Type.Object({
      product_id: Type.String({ minLength: 1, description: '用户提供或商品列表实际返回的商品 ID' }),
      page: Type.Optional(Type.Integer({ minimum: 1, maximum: 100000, description: '页码，默认 1' })),
      page_size: Type.Optional(Type.Integer({ minimum: 1, maximum: 100, description: '每页数量，默认 10' })),
    }, { additionalProperties: false }),
    execute: async (_toolCallId, { product_id, page, page_size }, signal) => {
      const result = await listProductSkus(product_id, { page, pageSize: page_size }, { signal });
      return { content: [{ type: 'text', text: JSON.stringify(result) }], details: result };
    },
  });
  pi.registerTool({
    name: 'list_products',
    label: '查询商品列表',
    description: '通过 Go API 按关键词分页查询真实商品；空关键词浏览全部商品。多词按后端规则匹配任意词，并非语义推荐。返回商品 ID、名称、价格及分页信息；不查询完整 SKU，不执行写操作。',
    parameters: Type.Object({
      keyword: Type.Optional(Type.String({ description: '商品关键词，省略或空字符串表示浏览全部商品' })),
      page: Type.Optional(Type.Integer({ minimum: 1, maximum: 100000, description: '页码，默认 1' })),
      page_size: Type.Optional(Type.Integer({ minimum: 1, maximum: 100, description: '每页数量，默认 10' })),
    }, { additionalProperties: false }),
    execute: async (_toolCallId, { keyword, page, page_size }, signal) => {
      const result = await listProducts({ keyword, page, pageSize: page_size }, { signal });
      return { content: [{ type: 'text', text: JSON.stringify(result) }], details: result };
    },
  });
  pi.registerTool({
    name: 'get_product',
    label: '查询商品详情',
    description: '根据已知商品 ID 查询 Go API 的真实商品详情。不能按商品名称搜索，不查询完整 SKU 列表，不执行写操作。',
    parameters: Type.Object({ product_id: Type.String({ minLength: 1, description: '已知的商品 ID' }) }),
    execute: async (_toolCallId, { product_id }, signal) => {
      const product = await getProduct(product_id, { signal });
      return { content: [{ type: 'text', text: JSON.stringify(product) }], details: { product } };
    },
  });
}
