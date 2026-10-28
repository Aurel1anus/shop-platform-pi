def list_products(keyword, page=1, page_size=2):
    params = {
        "keyword": keyword,
        "page": page,
        "page_size": page_size,
    }
    query = urlencode(params)
    url = f"{base_url}/products?{query}"
    request = Request(url, headers={"Accept": "application/json"})

    with urlopen(request, timeout=5) as response:
        result = json.loads(response.read())

    print(result)
    return result


