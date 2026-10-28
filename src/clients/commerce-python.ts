import { execFile } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const script = fileURLToPath(new URL('./commerce.py', import.meta.url));

interface ProductDetail {
  productId: string;
  name: string;
  price: string;
  [key: string]: unknown;
}

interface CallOptions {
  signal?: AbortSignal;
  timeoutMs?: number;
}

interface ProductListQuery {
  keyword?: string;
  page?: number;
  pageSize?: number;
}

interface ProductPage {
  items: ProductDetail[];
  total: number;
  page: number;
  page_size: number;
}

interface ProductSKU {
  skuId: string;
  productId: string;
  skuName: string;
  price: string;
  stockQuantity: number;
  stockStatus: string;
  specs: Record<string, string>;
}

interface SKUPage {
  items: ProductSKU[];
  total: number;
  page: number;
  page_size: number;
}

// 只负责跨语言调用；HTTP 请求、业务数据校验和错误转换由 Python 实现。
async function runPython<T>(
  args: string[], { signal, timeoutMs = 8000 }: CallOptions = {},
): Promise<T> {
  if (signal?.aborted) throw new Error('商品查询已取消');
  return new Promise<T>((resolve, reject) => {
    execFile(process.env.COMMERCE_PYTHON ?? 'python3', ['-B', script, ...args], {
      signal,
      timeout: timeoutMs,
      killSignal: 'SIGKILL',
      maxBuffer: 1024 * 1024,
      encoding: 'utf8',
      windowsHide: true,
    }, (error, stdout, stderr) => {
      if (signal?.aborted) return reject(new Error('商品查询已取消'));
      if (error?.code === 'ENOENT') return reject(new Error('找不到 Python，请检查 COMMERCE_PYTHON 配置'));
      if (error?.killed) return reject(new Error('商品查询超时，Python 进程已终止'));
      if (error) {
        // 只展示 Python 约定的错误 JSON，不透传命令、环境或 traceback。
        try {
          const result = JSON.parse(stderr);
          if (error.code === 1 && typeof result.error === 'string') return reject(new Error(result.error));
        } catch { /* 非协议错误统一处理。 */ }
        return reject(new Error('Python 商品客户端执行失败'));
      }
      try {
        resolve(JSON.parse(stdout));
      } catch {
        reject(new Error('Python 商品客户端返回的 JSON 无效'));
      }
    });
  });
}

export async function getProduct(
  productId: string, options: CallOptions = {},
): Promise<ProductDetail> {
  if (typeof productId !== 'string' || !productId.trim()) throw new Error('商品 ID 不能为空');
  return runPython<ProductDetail>(['--', productId], options);
}

export async function listProducts(
  { keyword = '', page = 1, pageSize = 10 }: ProductListQuery = {},
  options: CallOptions = {},
): Promise<ProductPage> {
  // 使用 --keyword=值，避免以 -- 开头的关键词被 argparse 当成另一个选项。
  return runPython<ProductPage>([
    '--list', `--keyword=${keyword}`, `--page=${page}`, `--page-size=${pageSize}`,
  ], options);
}

export async function listProductSkus(
  productId: string, { page = 1, pageSize = 10 }: { page?: number; pageSize?: number } = {},
  options: CallOptions = {},
): Promise<SKUPage> {
  if (typeof productId !== 'string' || !productId.trim()) throw new Error('商品 ID 不能为空');
  return runPython<SKUPage>([
    `--skus=${productId}`, `--page=${page}`, `--page-size=${pageSize}`,
  ], options);
}
