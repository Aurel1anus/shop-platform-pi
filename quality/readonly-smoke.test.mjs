import assert from 'node:assert/strict';
import { join } from 'node:path';
import { test } from 'node:test';
import { ModelRuntime } from '@earendil-works/pi-coding-agent';
import { createShopSession, runtimeDir } from '../src/pi-session.mjs';

// 使用开发演示数据 p_002；执行真实工具，不发送模型请求、不写业务数据。
test('只读商品真实服务验收（不调用模型）', async (t) => {
  const modelRuntime = await ModelRuntime.create({
    authPath: join(runtimeDir, 'check-auth.json'), modelsPath: null, refreshOnCreate: false,
  });
  const { session } = await createShopSession({ modelRuntime });
  t.after(() => session.dispose());
  const tools = new Map(session.agent.state.tools.map((tool) => [tool.name, tool]));
  for (const name of ['list_products', 'get_product', 'list_product_skus']) {
    assert.ok(session.getActiveToolNames().includes(name), `工具未启用：${name}`);
  }
  const call = async (name, args) => {
    const result = await tools.get(name).execute('readonly-smoke', args);
    const text = JSON.parse(result.content[0].text);
    assert.deepEqual(text, name === 'get_product' ? result.details.product : result.details);
    return text;
  };

  await t.test('商品查询→详情→规格，使用真实 ID 并保留价格及库存', async (caseTest) => {
    const list = await call('list_products', { keyword: 'Camera', page_size: 100 });
    const card = list.items.find((item) => item.productId === 'p_002');
    assert.ok(card, '未找到开发演示商品 p_002；请确认服务地址及演示数据');
    const product = await call('get_product', { product_id: card.productId });
    assert.equal(product.name, card.name);
    assert.equal(product.price, card.price);
    const skus = await call('list_product_skus', { product_id: card.productId });
    assert.ok(skus.total > 0, '开发演示商品应至少有一个 SKU');
    assert.ok(skus.items.length > 0);
    assert.ok(skus.total >= skus.items.length);
    for (const sku of skus.items) {
      assert.equal(sku.productId, card.productId);
      assert.equal(typeof sku.price, 'string');
      assert.ok(Number.isInteger(sku.stockQuantity) && sku.stockQuantity >= 0);
    }
    caseTest.diagnostic(JSON.stringify({
      productId: product.productId, name: product.name, price: product.price,
      skuTotal: skus.total, skuPage: skus.items,
    }));
  });

  await t.test('商品分页：每页一条，总数不随页码改变', async () => {
    const first = await call('list_products', { page: 1, page_size: 1 });
    const second = await call('list_products', { page: 2, page_size: 1 });
    assert.ok(first.total >= 2, '验收需要至少两个开发演示商品');
    assert.equal(second.total, first.total);
    assert.equal(first.items.length, 1);
    assert.equal(second.items.length, 1);
    assert.notEqual(first.items[0].productId, second.items[0].productId);
  });

  await t.test('无匹配：正常返回空列表，不补造商品', async () => {
    const result = await call('list_products', { keyword: '__readonly_smoke_no_match_7e3b09__' });
    assert.deepEqual(result.items, []);
    assert.equal(result.total, 0);
  });

  await t.test('不存在商品：详情明确失败，不返回假商品', async () => {
    await assert.rejects(call('get_product', { product_id: '__readonly_smoke_missing_7e3b09__' }), /商品不存在/);
  });

  await t.test('SKU 空页：保留总数，不将空结果等同商品不存在', async () => {
    const first = await call('list_product_skus', { product_id: 'p_002', page_size: 1 });
    assert.ok(first.total > 0 && first.total < 100000);
    const beyond = await call('list_product_skus', { product_id: 'p_002', page: 100000, page_size: 1 });
    assert.deepEqual(beyond.items, []);
    assert.equal(beyond.total, first.total);
    const missing = await call('list_product_skus', { product_id: '__readonly_smoke_missing_7e3b09__' });
    assert.deepEqual(missing.items, []);
    assert.equal(missing.total, 0);
  });
});
