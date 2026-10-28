import io
import json
import unittest
from contextlib import redirect_stderr, redirect_stdout
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError, URLError

from src.clients.commerce import CommerceError, get_product, list_products, list_product_skus, main


class CommerceTests(unittest.TestCase):
    def setUp(self):
        self.product = {"productId": "p_002", "name": "Y Camera Max", "price": "3499.00"}
        self.sku = {"skuId": "sku_002", "productId": "p_002", "skuName": "标准版",
                    "price": "3499.00", "stockQuantity": 32, "stockStatus": "in_stock",
                    "specs": {"版本": "标准版"}}
        self.opener = MagicMock()
        self.response = self.opener.open.return_value.__enter__.return_value
        self.response.read.return_value = json.dumps(self.product).encode()
        patcher = patch("src.clients.commerce.build_opener", return_value=self.opener)
        patcher.start()
        self.addCleanup(patcher.stop)

    def test_product_and_request(self):
        self.assertEqual(get_product("p_002", base_url="http://example.test/api/v1/"), self.product)
        request = self.opener.open.call_args.args[0]
        self.assertEqual(request.full_url, "http://example.test/api/v1/products/p_002")
        self.assertEqual(self.opener.open.call_args.kwargs["timeout"], 5)

    def test_invalid_inputs(self):
        for product_id in ("", "  ", None, 42):
            with self.subTest(product_id=product_id), self.assertRaisesRegex(CommerceError, "ID 不能为空"):
                get_product(product_id)
        for timeout in (0, -1, float("nan"), float("inf"), True, "5"):
            with self.subTest(timeout=timeout), self.assertRaisesRegex(CommerceError, "正数秒"):
                get_product("p_002", timeout=timeout)
        for url in ("", "ftp://example.test", "http://user:secret@example.test", "http://example.test?x=1"):
            with self.subTest(url=url), self.assertRaisesRegex(CommerceError, "地址无效"):
                get_product("p_002", base_url=url)
        self.opener.open.assert_not_called()

    def test_http_errors(self):
        for status, message in ((404, "商品不存在"), (500, "HTTP 500"), (302, "HTTP 302")):
            with self.subTest(status=status):
                self.opener.open.side_effect = HTTPError("unused", status, "unused", {}, None)
                with self.assertRaisesRegex(CommerceError, message):
                    get_product("p_002")

    def test_network_and_timeout(self):
        for error, message in ((URLError("offline"), "无法连接"),
                               (URLError(TimeoutError()), "超时"), (TimeoutError(), "超时")):
            with self.subTest(error=error):
                self.opener.open.side_effect = error
                with self.assertRaisesRegex(CommerceError, message):
                    get_product("p_002")
        self.opener.open.side_effect = None
        self.response.read.side_effect = TimeoutError()
        with self.assertRaisesRegex(CommerceError, "超时"):
            get_product("p_002")

    def test_invalid_response(self):
        for body, message in ((b"not JSON", "JSON 无效"), (b"\xff", "JSON 无效"),
                              (b"null", "数据格式"), (b"[]", "数据格式"),
                              (json.dumps({**self.product, "productId": "wrong"}).encode(), "数据格式"),
                              (json.dumps({**self.product, "price": 3499}).encode(), "数据格式")):
            with self.subTest(body=body), self.assertRaisesRegex(CommerceError, message):
                self.response.read.return_value = body
                get_product("p_002")

    def test_environment_url_and_encoded_id(self):
        product_id = "id/with?query"
        self.response.read.return_value = json.dumps({**self.product, "productId": product_id}).encode()
        with patch.dict("os.environ", {"COMMERCE_API_BASE_URL": "http://example.test/api/v1"}):
            get_product(product_id)
        self.assertEqual(self.opener.open.call_args.args[0].full_url,
                         "http://example.test/api/v1/products/id%2Fwith%3Fquery")

    def test_cli_output_and_exit_status(self):
        stdout, stderr = io.StringIO(), io.StringIO()
        with patch("sys.argv", ["commerce.py", "p_002"]), redirect_stdout(stdout), redirect_stderr(stderr):
            self.assertEqual(main(), 0)
        self.assertEqual(json.loads(stdout.getvalue()), self.product)
        self.assertEqual(stderr.getvalue(), "")
        self.opener.open.side_effect = URLError("private connection details")
        stdout, stderr = io.StringIO(), io.StringIO()
        with patch("sys.argv", ["commerce.py", "p_002"]), redirect_stdout(stdout), redirect_stderr(stderr):
            self.assertEqual(main(), 1)
        self.assertEqual(stdout.getvalue(), "")
        self.assertEqual(json.loads(stderr.getvalue()), {"error": "无法连接商品服务"})

    def test_list_products_and_empty_page(self):
        payload = {"items": [self.product], "total": 2, "page": 2, "page_size": 1}
        self.response.read.return_value = json.dumps(payload).encode()
        with patch.dict("os.environ", {"COMMERCE_API_BASE_URL": "http://example.test/api/v1/"}):
            self.assertEqual(list_products("Camera & Mouse", page=2, page_size=1), payload)
        self.assertEqual(self.opener.open.call_args.args[0].full_url,
                         "http://example.test/api/v1/products?keyword=Camera+%26+Mouse&page=2&page_size=1")
        empty = {"items": [], "total": 0, "page": 1, "page_size": 10}
        self.response.read.return_value = json.dumps(empty).encode()
        self.assertEqual(list_products(), empty)

    def test_list_cli_output_and_errors(self):
        payload = {"items": [self.product], "total": 2, "page": 2, "page_size": 1}
        self.response.read.return_value = json.dumps(payload).encode()
        stdout, stderr = io.StringIO(), io.StringIO()
        argv = ["commerce.py", "--list", "--keyword", "Camera & Mouse", "--page", "2",
                "--page-size", "1", "--base-url", "http://example.test/api/v1", "--timeout", "3"]
        with patch("sys.argv", argv), redirect_stdout(stdout), redirect_stderr(stderr):
            self.assertEqual(main(), 0)
        self.assertEqual(json.loads(stdout.getvalue()), payload)
        self.assertEqual(stderr.getvalue(), "")
        self.assertEqual(self.opener.open.call_args.args[0].full_url,
                         "http://example.test/api/v1/products?keyword=Camera+%26+Mouse&page=2&page_size=1")
        self.assertEqual(self.opener.open.call_args.kwargs["timeout"], 3)

        empty = {"items": [], "total": 0, "page": 1, "page_size": 10}
        self.response.read.return_value = json.dumps(empty).encode()
        stdout, stderr = io.StringIO(), io.StringIO()
        with patch("sys.argv", ["commerce.py", "--list"]), redirect_stdout(stdout), redirect_stderr(stderr):
            self.assertEqual(main(), 0)
        self.assertEqual(json.loads(stdout.getvalue()), empty)
        self.assertEqual(stderr.getvalue(), "")

        for options, error, message in [
            (["--page-size", "0"], None, "每页数量必须是 1 到 100 的整数"),
            ([], URLError("private connection details"), "无法连接商品服务"),
            ([], HTTPError("unused", 500, "unused", {}, None), "商品服务请求失败（HTTP 500）"),
        ]:
            self.opener.open.reset_mock()
            self.opener.open.side_effect = error
            stdout, stderr = io.StringIO(), io.StringIO()
            with self.subTest(options=options, error=error), patch("sys.argv", ["commerce.py", "--list", *options]), redirect_stdout(stdout), redirect_stderr(stderr):
                self.assertEqual(main(), 1)
            self.assertEqual(stdout.getvalue(), "")
            self.assertEqual(json.loads(stderr.getvalue()), {"error": message})
            if options:
                self.opener.open.assert_not_called()

    def test_cli_usage_errors(self):
        for options in ([], ["--list", "p_002"], ["--list", "--page", "abc"],
                        ["p_002", "--keyword", "Camera"], ["--skus"],
                        ["--skus=p_002", "--list"], ["--skus=p_002", "p_002"],
                        ["--skus=p_002", "--keyword", "Camera"], ["p_002", "--page", "2"]):
            stdout, stderr = io.StringIO(), io.StringIO()
            with self.subTest(options=options), patch("sys.argv", ["commerce.py", *options]), redirect_stdout(stdout), redirect_stderr(stderr):
                with self.assertRaises(SystemExit) as raised:
                    main()
            self.assertEqual(raised.exception.code, 2)
            self.assertEqual(stdout.getvalue(), "")
            self.assertIn("usage:", stderr.getvalue())
        self.opener.open.assert_not_called()

    def test_list_invalid_inputs(self):
        cases = [({"keyword": None}, "关键词"), ({"keyword": 1}, "关键词")]
        cases += [({"page": value}, "页码") for value in (True, 0, -1, 100001, 1.5, "1")]
        cases += [({"page_size": value}, "每页数量") for value in (False, 0, -1, 101, 1.5, "2")]
        cases += [({"timeout": 0}, "正数秒"), ({"base_url": "ftp://example.test"}, "地址无效")]
        for kwargs, message in cases:
            with self.subTest(kwargs=kwargs), self.assertRaisesRegex(CommerceError, message):
                list_products(**kwargs)
        self.opener.open.assert_not_called()

    def test_list_failures(self):
        for error, message in ((HTTPError("unused", 404, "unused", {}, None), "HTTP 404"),
                               (HTTPError("unused", 500, "unused", {}, None), "HTTP 500"),
                               (URLError("offline"), "无法连接"), (TimeoutError(), "超时")):
            self.opener.open.side_effect = error
            with self.subTest(error=error), self.assertRaisesRegex(CommerceError, message):
                list_products()
        self.opener.open.side_effect = None
        self.response.read.return_value = b"not JSON"
        with self.assertRaisesRegex(CommerceError, "JSON 无效"):
            list_products()

    def test_list_invalid_response(self):
        valid = {"items": [self.product], "total": 1, "page": 1, "page_size": 10}
        invalid = [None, [], {}, {**valid, "items": None}, {**valid, "total": True},
                   {**valid, "total": -1}, {**valid, "page": True}, {**valid, "page": 2},
                   {**valid, "page_size": 1}, {**valid, "items": [None]},
                   {**valid, "items": [{**self.product, "price": 3499}]}]
        for payload in invalid:
            self.response.read.return_value = json.dumps(payload).encode()
            with self.subTest(payload=payload), self.assertRaisesRegex(CommerceError, "数据格式"):
                list_products()

    def test_skus_request_pagination_and_empty(self):
        product_id = "id/with?query"
        payload = {"items": [{**self.sku, "productId": product_id}], "total": 2, "page": 2, "page_size": 1}
        self.response.read.return_value = json.dumps(payload).encode()
        self.assertEqual(list_product_skus(product_id, page=2, page_size=1,
                                           base_url="http://example.test/api/v1/"), payload)
        self.assertEqual(self.opener.open.call_args.args[0].full_url,
                         "http://example.test/api/v1/products/id%2Fwith%3Fquery/skus?page=2&page_size=1")
        for total in (0, 2):
            empty = {"items": [], "total": total, "page": 3, "page_size": 1}
            self.response.read.return_value = json.dumps(empty).encode()
            self.assertEqual(list_product_skus("p_002", page=3, page_size=1), empty)

    def test_skus_invalid_inputs(self):
        for product_id in ("", "  ", None, 42):
            with self.subTest(product_id=product_id), self.assertRaisesRegex(CommerceError, "ID 不能为空"):
                list_product_skus(product_id)
        cases = [({"page": value}, "页码") for value in (True, 0, 100001, 1.5)]
        cases += [({"page_size": value}, "每页数量") for value in (False, 0, 101, "2")]
        cases += [({"timeout": 0}, "正数秒"), ({"base_url": "ftp://example.test"}, "地址无效")]
        for kwargs, message in cases:
            with self.subTest(kwargs=kwargs), self.assertRaisesRegex(CommerceError, message):
                list_product_skus("p_002", **kwargs)
        self.opener.open.assert_not_called()

    def test_skus_invalid_response_and_ownership(self):
        valid = {"items": [self.sku], "total": 1, "page": 1, "page_size": 10}
        bad_skus = [None, {}, {**self.sku, "productId": "other_product"},
                    {**self.sku, "skuId": ""}, {**self.sku, "skuName": None},
                    {**self.sku, "price": 3499}, {**self.sku, "stockQuantity": True},
                    {**self.sku, "stockQuantity": -1}, {**self.sku, "stockStatus": None},
                    {**self.sku, "specs": None}, {**self.sku, "specs": {"版本": 1}}]
        invalid = [None, [], {}, {**valid, "items": None}, {**valid, "total": True},
                   {**valid, "total": -1}, {**valid, "page": 2}, {**valid, "page_size": True}]
        invalid += [{**valid, "items": [sku]} for sku in bad_skus]
        for payload in invalid:
            self.response.read.return_value = json.dumps(payload).encode()
            with self.subTest(payload=payload), self.assertRaisesRegex(CommerceError, "数据格式"):
                list_product_skus("p_002")
        zero_stock = {**valid, "items": [{**self.sku, "stockQuantity": 0, "stockStatus": "out_of_stock", "specs": {}}]}
        self.response.read.return_value = json.dumps(zero_stock).encode()
        self.assertEqual(list_product_skus("p_002"), zero_stock)

    def test_skus_request_failures(self):
        for error, message in ((HTTPError("unused", 404, "unused", {}, None), "HTTP 404"),
                               (HTTPError("unused", 500, "unused", {}, None), "HTTP 500"),
                               (HTTPError("unused", 302, "unused", {}, None), "HTTP 302"),
                               (URLError("offline"), "无法连接"), (TimeoutError(), "超时")):
            self.opener.open.side_effect = error
            with self.subTest(error=error), self.assertRaisesRegex(CommerceError, message):
                list_product_skus("p_002")
        self.opener.open.side_effect = None
        self.response.read.return_value = b"not JSON"
        with self.assertRaisesRegex(CommerceError, "JSON 无效"):
            list_product_skus("p_002")

    def test_skus_cli(self):
        payload = {"items": [self.sku], "total": 2, "page": 2, "page_size": 1}
        self.response.read.return_value = json.dumps(payload).encode()
        stdout, stderr = io.StringIO(), io.StringIO()
        options = ["commerce.py", "--skus=p_002", "--page", "2", "--page-size", "1",
                   "--base-url", "http://example.test/api/v1", "--timeout", "3"]
        with patch("sys.argv", options), redirect_stdout(stdout), redirect_stderr(stderr):
            self.assertEqual(main(), 0)
        self.assertEqual(json.loads(stdout.getvalue()), payload)
        self.assertEqual(stderr.getvalue(), "")
        self.assertEqual(self.opener.open.call_args.args[0].full_url,
                         "http://example.test/api/v1/products/p_002/skus?page=2&page_size=1")
        self.assertEqual(self.opener.open.call_args.kwargs["timeout"], 3)
        for suffix, error in [(["--page-size", "0"], None), ([], URLError("private details"))]:
            self.opener.open.side_effect = error
            stdout, stderr = io.StringIO(), io.StringIO()
            with patch("sys.argv", ["commerce.py", "--skus=p_002", *suffix]), redirect_stdout(stdout), redirect_stderr(stderr):
                self.assertEqual(main(), 1)
            self.assertEqual(stdout.getvalue(), "")
            self.assertIn("error", json.loads(stderr.getvalue()))


if __name__ == "__main__":
    unittest.main()
