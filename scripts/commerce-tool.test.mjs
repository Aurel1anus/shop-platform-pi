import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { join } from 'node:path';
import { test } from 'node:test';
import { ModelRuntime } from '@earendil-works/pi-coding-agent';
import { getProduct, listProducts, listProductSkus } from '../src/clients/commerce-python.ts';
import { createShopSession, runtimeDir } from '../src/pi-session.mjs';

test('pi 工具 → Python → 模拟 HTTP 服务', async (t) => {
  const product = { productId: 'p_002', name: '测试商品', price: '12.50' };
  let listParams;
  let skuParams;
  const server = createServer((req, res) => {
    const url = new URL(req.url, 'http://localhost');
    const skuMatch = url.pathname.match(/^\/api\/v1\/products\/(.+)\/skus$/);
    if (skuMatch) {
      const productId = decodeURIComponent(skuMatch[1]);
      skuParams = { productId, ...Object.fromEntries(url.searchParams) };
      if (productId === 'slow') return;
      res.setHeader('Content-Type', 'application/json');
      if (productId === 'http-missing') return res.writeHead(404).end('{}');
      if (productId === 'failure') return res.writeHead(500).end('{}');
      if (productId === 'invalid') return res.end('not JSON');
      const skus = ['standard', 'large'].map((version, index) => ({
        productId: productId === 'wrong-owner' ? 'other-product' : productId,
        skuId: `sku_${version}`, skuName: version, price: '12.50',
        stockQuantity: index === 0 ? 32 : 0,
        stockStatus: index === 0 ? 'in_stock' : 'out_of_stock', specs: { 版本: version },
      }));
      const page = Number(url.searchParams.get('page'));
      const pageSize = Number(url.searchParams.get('page_size'));
      return res.end(JSON.stringify({
        items: productId === 'empty' ? [] : skus.slice((page - 1) * pageSize, page * pageSize),
        total: productId === 'empty' ? 0 : skus.length, page, page_size: pageSize,
      }));
    }
    if (url.pathname === '/api/v1/products') {
      listParams = Object.fromEntries(url.searchParams);
      const keyword = url.searchParams.get('keyword');
      if (keyword === 'slow') return;
      res.setHeader('Content-Type', 'application/json');
      if (keyword === 'missing') return res.writeHead(404).end('{}');
      if (keyword === 'failure') return res.writeHead(500).end('{}');
      if (keyword === 'invalid') return res.end('not JSON');
      if (keyword === 'malformed') return res.end(JSON.stringify({ items: null, total: 0 }));
      return res.end(JSON.stringify({
        items: keyword === 'empty' ? [] : [product],
        total: keyword === 'empty' ? 0 : 2,
        page: Number(url.searchParams.get('page')),
        page_size: Number(url.searchParams.get('page_size')),
      }));
    }
    const id = decodeURIComponent(req.url.split('/').at(-1));
    if (id === 'slow') return; // 由适配层超时或取消终止 Python 请求。
    res.setHeader('Content-Type', 'application/json');
    if (id === 'missing') {
      res.writeHead(404).end('{}');
    } else if (id === 'failure') {
      res.writeHead(500).end('{}');
    } else if (id === 'invalid') {
      res.end('not JSON');
    } else {
      res.end(JSON.stringify({ ...product, productId: id }));
    }
  });
  const oldUrl = process.env.COMMERCE_API_BASE_URL;
  const oldPython = process.env.COMMERCE_PYTHON;
  const oldNoProxy = process.env.no_proxy;
  // 测试仅访问本机；隔离机器代理，确保断开服务时得到连接失败。
  process.env.no_proxy = '*';
  t.after(() => {
    if (oldUrl === undefined) delete process.env.COMMERCE_API_BASE_URL;
    else process.env.COMMERCE_API_BASE_URL = oldUrl;
    if (oldPython === undefined) delete process.env.COMMERCE_PYTHON;
    else process.env.COMMERCE_PYTHON = oldPython;
    if (oldNoProxy === undefined) delete process.env.no_proxy;
    else process.env.no_proxy = oldNoProxy;
    server.closeAllConnections();
    server.close();
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  process.env.COMMERCE_API_BASE_URL = `http://127.0.0.1:${server.address().port}/api/v1`;

  const modelRuntime = await ModelRuntime.create({
    authPath: join(runtimeDir, 'check-auth.json'), modelsPath: null, refreshOnCreate: false,
  });
  const { session } = await createShopSession({ modelRuntime });
  t.after(() => session.dispose());
  assert.ok(session.getActiveToolNames().includes('get_product'));
  assert.ok(session.getActiveToolNames().includes('list_products'));
  assert.ok(session.getActiveToolNames().includes('list_product_skus'));
  const tool = session.agent.state.tools.find((item) => item.name === 'get_product');
  const listTool = session.agent.state.tools.find((item) => item.name === 'list_products');
  const skuTool = session.agent.state.tools.find((item) => item.name === 'list_product_skus');

  await t.test('真实注册的工具返回文本与结构化商品', async () => {
    const result = await tool.execute('test-product', { product_id: 'p_002' });
    assert.deepEqual(result.details.product, product);
    assert.deepEqual(JSON.parse(result.content[0].text), product);
  });
  await t.test('不存在商品、后端失败与无效数据不会返回成功', async () => {
    for (const [id, message] of [['missing', /商品不存在/], ['failure', /HTTP 500/], ['invalid', /JSON 无效/]]) {
      await assert.rejects(tool.execute('test-error', { product_id: id }), message);
    }
    await assert.rejects(getProduct('  '), /ID 不能为空/);
    // -- 确保商品 ID 不能被 Python 当作命令行选项。
    assert.equal((await getProduct('--help')).productId, '--help');
  });
  await t.test('列表适配经 Python 传递关键词与分页，返回完整结果', async () => {
    const result = await listProducts({ keyword: 'Camera & Mouse 中文', page: 2, pageSize: 1 });
    assert.deepEqual(listParams, { keyword: 'Camera & Mouse 中文', page: '2', page_size: '1' });
    assert.deepEqual(result, { items: [product], total: 2, page: 2, page_size: 1 });
    await listProducts();
    assert.deepEqual(listParams, { keyword: '', page: '1', page_size: '10' });
    for (const keyword of ['--help', 'x&y=$(echo unsafe);']) {
      await listProducts({ keyword });
      assert.equal(listParams.keyword, keyword);
    }
    assert.deepEqual(await listProducts({ keyword: 'empty' }), {
      items: [], total: 0, page: 1, page_size: 10,
    });
  });
  await t.test('注册的列表工具映射分页参数并返回文本与结构化结果', async () => {
    const result = await listTool.execute('test-list', {
      keyword: 'Camera & Mouse 中文', page: 2, page_size: 1,
    });
    assert.deepEqual(listParams, { keyword: 'Camera & Mouse 中文', page: '2', page_size: '1' });
    const expected = { items: [product], total: 2, page: 2, page_size: 1 };
    assert.deepEqual(result.details, expected);
    assert.deepEqual(JSON.parse(result.content[0].text), expected);
    const defaults = await listTool.execute('test-list-defaults', {});
    assert.deepEqual(listParams, { keyword: '', page: '1', page_size: '10' });
    assert.equal(defaults.details.page, 1);
    const empty = await listTool.execute('test-list-empty', { keyword: 'empty' });
    assert.deepEqual(empty.details, { items: [], total: 0, page: 1, page_size: 10 });
    assert.deepEqual(JSON.parse(empty.content[0].text), empty.details);
  });
  await t.test('列表输入错误与后端错误经 Python 转为失败', async () => {
    await assert.rejects(listProducts({ pageSize: 0 }), /每页数量/);
    await assert.rejects(listProducts({ page: 0 }), /页码/);
    await assert.rejects(listTool.execute('test-list-page-error', { page_size: 0 }), /每页数量/);
    for (const [keyword, message] of [
      ['missing', /HTTP 404/], ['failure', /HTTP 500/],
      ['invalid', /JSON 无效/], ['malformed', /数据格式/],
    ]) {
      await assert.rejects(listProducts({ keyword }), message);
      await assert.rejects(listTool.execute('test-list-error', { keyword }), message);
    }
  });
  await t.test('规格工具返回各 SKU 的价格、库存及分页，保留空页', async () => {
    const result = await skuTool.execute('test-sku', { product_id: 'p_002', page: 2, page_size: 1 });
    assert.deepEqual(skuParams, { productId: 'p_002', page: '2', page_size: '1' });
    assert.equal(result.details.total, 2);
    assert.equal(result.details.items.length, 1);
    assert.deepEqual(result.details.items[0], {
      productId: 'p_002', skuId: 'sku_large', skuName: 'large', price: '12.50',
      stockQuantity: 0, stockStatus: 'out_of_stock', specs: { 版本: 'large' },
    });
    assert.deepEqual(JSON.parse(result.content[0].text), result.details);
    const defaults = await skuTool.execute('test-sku-defaults', { product_id: 'p_002' });
    assert.deepEqual(skuParams, { productId: 'p_002', page: '1', page_size: '10' });
    assert.equal(defaults.details.items.length, 2);
    const beyond = await skuTool.execute('test-sku-beyond', { product_id: 'p_002', page: 3, page_size: 1 });
    assert.deepEqual(beyond.details, { items: [], total: 2, page: 3, page_size: 1 });
    const empty = await skuTool.execute('test-sku-empty', { product_id: 'empty' });
    assert.deepEqual(empty.details, { items: [], total: 0, page: 1, page_size: 10 });
    for (const productId of ['--help', 'id/with?query&中文']) {
      assert.equal((await listProductSkus(productId)).items[0].productId, productId);
      assert.equal(skuParams.productId, productId);
    }
  });
  await t.test('规格工具拒绝错误归属、非法输入与服务失败', async () => {
    await assert.rejects(listProductSkus('  '), /ID 不能为空/);
    await assert.rejects(skuTool.execute('test-sku-page', { product_id: 'p_002', page: 0 }), /页码/);
    await assert.rejects(skuTool.execute('test-sku-size', { product_id: 'p_002', page_size: 101 }), /每页数量/);
    for (const [productId, message] of [
      ['http-missing', /HTTP 404/], ['failure', /HTTP 500/],
      ['invalid', /JSON 无效/], ['wrong-owner', /数据格式/],
    ]) {
      await assert.rejects(skuTool.execute('test-sku-error', { product_id: productId }), message);
    }
  });
  await t.test('整体超时会终止 Python 子进程', async () => {
    await assert.rejects(getProduct('slow', { timeoutMs: 500 }), /超时.*进程已终止/);
    await assert.rejects(listProducts({ keyword: 'slow' }, { timeoutMs: 500 }), /超时.*进程已终止/);
    await assert.rejects(listProductSkus('slow', {}, { timeoutMs: 500 }), /超时.*进程已终止/);
  });
  await t.test('取消信号传给子进程', async () => {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 200);
    try {
      await assert.rejects(tool.execute('test-cancel', { product_id: 'slow' }, controller.signal), /已取消/);
    } finally {
      clearTimeout(timer);
    }
    await assert.rejects(getProduct('p_002', { signal: AbortSignal.abort() }), /已取消/);
    const listController = new AbortController();
    const listTimer = setTimeout(() => listController.abort(), 200);
    try {
      await assert.rejects(listTool.execute('test-list-cancel', { keyword: 'slow' }, listController.signal), /已取消/);
    } finally {
      clearTimeout(listTimer);
    }
    await assert.rejects(listTool.execute('test-list-aborted', {}, AbortSignal.abort()), /已取消/);
    const skuController = new AbortController();
    const skuTimer = setTimeout(() => skuController.abort(), 200);
    try {
      await assert.rejects(skuTool.execute('test-sku-cancel', { product_id: 'slow' }, skuController.signal), /已取消/);
    } finally {
      clearTimeout(skuTimer);
    }
    await assert.rejects(skuTool.execute('test-sku-aborted', { product_id: 'p_002' }, AbortSignal.abort()), /已取消/);
  });
  await t.test('Python 不可用时明确报错', async () => {
    process.env.COMMERCE_PYTHON = '/nonexistent-commerce-python';
    try {
      await assert.rejects(getProduct('p_002'), /找不到 Python/);
      await assert.rejects(listProducts(), /找不到 Python/);
      await assert.rejects(listTool.execute('test-list-python', {}), /找不到 Python/);
      await assert.rejects(skuTool.execute('test-sku-python', { product_id: 'p_002' }), /找不到 Python/);
    } finally {
      if (oldPython === undefined) delete process.env.COMMERCE_PYTHON;
      else process.env.COMMERCE_PYTHON = oldPython;
    }
  });
  await t.test('连接失败', async () => {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
    await assert.rejects(getProduct('p_002'), /无法连接商品服务/);
    await assert.rejects(listProducts(), /无法连接商品服务/);
    await assert.rejects(listTool.execute('test-list-offline', {}), /无法连接商品服务/);
    await assert.rejects(skuTool.execute('test-sku-offline', { product_id: 'p_002' }), /无法连接商品服务/);
  });
});
