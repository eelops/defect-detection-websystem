import requests

API_URL = "http://localhost:8000/api/external/report"
API_KEY = "please-change-this-key"

# 1x1 PNG
IMAGE_BASE64 = (
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8"
    "/w8AAn8B9oN8WQAAAABJRU5ErkJggg=="
)

payload = {
    "image_base64": IMAGE_BASE64,
    "defect_type": "裂纹",
    "severity": "严重",
    "location": "零件A-区域3",
    "confidence": 0.95,
    "description": "模型自动检测到裂纹缺陷",
    "source": "示例检测模型",
}

headers = {
    "Content-Type": "application/json",
    "X-API-Key": API_KEY,
}

response = requests.post(API_URL, json=payload, headers=headers, timeout=30)
print("status:", response.status_code)
print("response:", response.json())
