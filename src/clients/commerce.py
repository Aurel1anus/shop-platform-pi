"""通过公开 Go API 读取商品；不写业务表，不调用模型。"""

import argparse
import json
import math
import os
import socket
import sys
from http.client import HTTPException
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlencode, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener


class CommerceError(Exception):
    """可安全展示给调用方的商品查询错误。"""


class _NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def _request_json(path: str, *, base_url: str | None, timeout: float,
                  not_found_message: str | None = None):
    if not isinstance(timeout, (int, float)) or isinstance(timeout, bool) or not math.isfinite(timeout) or timeout <= 0:
        raise CommerceError("查询超时必须是正数秒")
    if base_url is None:
        base_url = os.environ.get("COMMERCE_API_BASE_URL", "http://127.0.0.1:8081/api/v1")
    try:
        parsed = urlsplit(base_url)
        if (parsed.scheme not in ("http", "https") or not parsed.hostname
                or parsed.username is not None or parsed.password is not None
                or parsed.query or parsed.fragment):
            raise ValueError
        url = f"{base_url.rstrip('/')}{path}"
        request = Request(url, headers={"Accept": "application/json"})
    except (ValueError, TypeError, AttributeError):
        raise CommerceError("商品服务地址无效") from None

    # 不跟随重定向；timeout 是同步请求的 socket 等待超时，并非整次调用总时限。
    try:
        with build_opener(_NoRedirect()).open(request, timeout=timeout) as response:
            body = response.read()
    except HTTPError as error:
        status = error.code
        error.close()
        if status == 404 and not_found_message is not None:
            raise CommerceError(not_found_message) from None
        raise CommerceError(f"商品服务请求失败（HTTP {status}）") from None
    except (TimeoutError, socket.timeout):
        raise CommerceError("商品查询超时") from None
    except URLError as error:
        if isinstance(error.reason, TimeoutError):
            raise CommerceError("商品查询超时") from None
        raise CommerceError("无法连接商品服务") from None
    except (OSError, ValueError, HTTPException):
        raise CommerceError("无法连接商品服务") from None

    try:
        return json.loads(body)
    except (ValueError, UnicodeError):
        raise CommerceError("商品服务返回的 JSON 无效") from None


def get_product(product_id: str, *, base_url: str | None = None, timeout: float = 5) -> dict:
    if not isinstance(product_id, str) or not product_id.strip():
        raise CommerceError("商品 ID 不能为空")
    product = _request_json(
        f"/products/{quote(product_id, safe='')}",
        base_url=base_url, timeout=timeout, not_found_message="商品不存在",
    )
    if (not isinstance(product, dict) or product.get("productId") != product_id
            or not isinstance(product.get("name"), str)
            or not isinstance(product.get("price"), str)):
        raise CommerceError("商品服务返回的数据格式不正确")
    return product


def list_products(keyword: str = "", *, page: int = 1, page_size: int = 10,
                  base_url: str | None = None, timeout: float = 5) -> dict:
    """按关键词分页浏览商品；空列表表示无匹配，服务错误通过异常报告。"""
    if not isinstance(keyword, str):
        raise CommerceError("商品关键词必须是字符串")
    _validate_pagination(page, page_size)

    query = urlencode({"keyword": keyword, "page": page, "page_size": page_size})
    result = _request_json(f"/products?{query}", base_url=base_url, timeout=timeout)
    _validate_page_result(result, page, page_size)
    for product in result["items"]:
        if (not isinstance(product, dict)
                or not isinstance(product.get("productId"), str) or not product["productId"]
                or not isinstance(product.get("name"), str)
                or not isinstance(product.get("price"), str)):
            raise CommerceError("商品服务返回的数据格式不正确")
    return result


def _validate_pagination(page: int, page_size: int) -> None:
    # bool 是 int 的子类；分页参数不接受 True/False。
    if type(page) is not int or not 1 <= page <= 100000:
        raise CommerceError("页码必须是 1 到 100000 的整数")
    if type(page_size) is not int or not 1 <= page_size <= 100:
        raise CommerceError("每页数量必须是 1 到 100 的整数")


def _validate_page_result(result, page: int, page_size: int) -> None:
    if (not isinstance(result, dict)
            or not isinstance(result.get("items"), list)
            or type(result.get("total")) is not int or result["total"] < 0
            or type(result.get("page")) is not int or result["page"] != page
            or type(result.get("page_size")) is not int or result["page_size"] != page_size):
        raise CommerceError("商品服务返回的数据格式不正确")


def list_product_skus(product_id: str, *, page: int = 1, page_size: int = 10,
                      base_url: str | None = None, timeout: float = 5) -> dict:
    """读取指定商品的分页规格；空结果不能区分无规格与商品不存在。"""
    if not isinstance(product_id, str) or not product_id.strip():
        raise CommerceError("商品 ID 不能为空")
    _validate_pagination(page, page_size)
    query = urlencode({"page": page, "page_size": page_size})
    result = _request_json(
        f"/products/{quote(product_id, safe='')}/skus?{query}",
        base_url=base_url, timeout=timeout,
    )
    _validate_page_result(result, page, page_size)
    for sku in result["items"]:
        if (not isinstance(sku, dict) or sku.get("productId") != product_id
                or not isinstance(sku.get("skuId"), str) or not sku["skuId"]
                or not isinstance(sku.get("skuName"), str)
                or not isinstance(sku.get("price"), str)
                or type(sku.get("stockQuantity")) is not int or sku["stockQuantity"] < 0
                or not isinstance(sku.get("stockStatus"), str)
                or not isinstance(sku.get("specs"), dict)
                or not all(isinstance(key, str) and isinstance(value, str)
                           for key, value in sku["specs"].items())):
            raise CommerceError("商品服务返回的数据格式不正确")
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description="查询 Go API 的商品详情、列表或规格")
    query = parser.add_mutually_exclusive_group(required=True)
    query.add_argument("product_id", nargs="?", help="查询详情的商品 ID")
    query.add_argument("--list", action="store_true", help="查询商品列表")
    query.add_argument("--skus", metavar="PRODUCT_ID", help="查询指定商品的规格列表")
    parser.add_argument("--keyword", help="列表关键词，默认浏览全部商品")
    parser.add_argument("--page", type=int, help="商品/规格列表页码，默认 1")
    parser.add_argument("--page-size", type=int, help="商品/规格列表每页数量，默认 10")
    parser.add_argument("--base-url", default=None)
    parser.add_argument("--timeout", type=float, default=5)
    args = parser.parse_args()
    if args.keyword is not None and not args.list:
        parser.error("--keyword 只能用于 --list")
    if not args.list and args.skus is None and any(value is not None for value in (args.page, args.page_size)):
        parser.error("--page 和 --page-size 只能用于 --list 或 --skus")
    try:
        if args.list:
            result = list_products(
                keyword=args.keyword if args.keyword is not None else "",
                page=args.page if args.page is not None else 1,
                page_size=args.page_size if args.page_size is not None else 10,
                base_url=args.base_url, timeout=args.timeout,
            )
        elif args.skus is not None:
            result = list_product_skus(
                args.skus,
                page=args.page if args.page is not None else 1,
                page_size=args.page_size if args.page_size is not None else 10,
                base_url=args.base_url, timeout=args.timeout,
            )
        else:
            result = get_product(args.product_id, base_url=args.base_url, timeout=args.timeout)
    except CommerceError as error:
        print(json.dumps({"error": str(error)}, ensure_ascii=False), file=sys.stderr)
        return 1
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
