import torch
import numpy as np
from PIL import Image
import io
import requests

class HTTPUploadImage:
    @classmethod
    def INPUT_TYPES(s):
        return {
            "required": {
                "images": ("IMAGE",),
                "filename_prefix": ("STRING", {"default": "img"}),
            }
        }

    RETURN_TYPES = ()
    FUNCTION = "upload"
    OUTPUT_NODE = True
    CATEGORY = "api"

    def upload(self, images, filename_prefix="img"):
        for idx, image in enumerate(images):
            # Convert [H, W, C] float32 (0~1) → uint8 PIL
            img_array = (255.0 * image.cpu().numpy()).clip(0, 255).astype(np.uint8)
            pil_img = Image.fromarray(img_array)

            # Encode to PNG in memory
            buf = io.BytesIO()
            pil_img.save(buf, format='PNG')
            buf.seek(0)

            filename = f"{filename_prefix}_{idx}.png"

            try:
                files = {'image': (filename, buf, 'image/png')}
                resp = requests.post(
                    "http://47.82.92.91:38080/api/v1/upload",
                    files=files,
                    timeout=10
                )
                if resp.status_code == 200:
                    print(f"✅ Uploaded {filename}")
                else:
                    print(f"❌ Upload failed: {resp.status_code} {resp.text}")
            except Exception as e:
                print(f"⚠️ Upload error: {e}")

        return {}

NODE_CLASS_MAPPINGS = {"HTTPUploadImage": HTTPUploadImage}
NODE_DISPLAY_NAME_MAPPINGS = {"HTTPUploadImage": "HTTP Upload Image"}
