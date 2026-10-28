import type { ExtensionAPI } from '@earendil-works/pi-coding-agent';
import { Type } from 'typebox';

export default function status(pi: ExtensionAPI) {
  pi.registerTool({
    name: 'shop_status',
    label: '购物助手接入状态',
    description: '查询本地 pi 接入及业务插件的实现状态；不连接后端，不执行购物操作。',
    parameters: Type.Object({}),
    execute: async () => {
      const details = {
        pi: 'configured',
        backendUrl: process.env.COMMERCE_API_BASE_URL ?? 'http://127.0.0.1:8081/api/v1',
        backendConnection: 'not_checked',
        commerce: 'product_read_only',
        commerceTools: ['get_product', 'list_products', 'list_product_skus'],
        memory: 'placeholder',
        rag: 'placeholder',
      };
      return { content: [{ type: 'text', text: JSON.stringify(details) }], details };
    },
  });
}
