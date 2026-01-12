import json
import random
import time
import requests

def func(cfg: float, steps: int, style_suffix: str = ""):

    # === Configuration ===
    COMFYUI_URL = "http://127.0.0.1:8188"
    WORKFLOW_PATH = "chroma_workflow_api.json"
    PROMPTS_PATH = "prompts.json"
    POSITIVE_NODE_ID = "748"
    NEGATIVE_NODE_ID = "749"
    SEED_NODE_ID = "718"
    SAVE_NODE_ID = "740"

    # === Load workflow template ===
    with open(WORKFLOW_PATH, "r", encoding="utf-8") as f:
        base_workflow = json.load(f)

    # Ensure batch_size = 3 (limited VRAM)
    if "737" in base_workflow and "inputs" in base_workflow["737"]:
        base_workflow["737"]["inputs"]["batch_size"] = 3

    # Adjust cfg
    assert "694" in base_workflow, "CFG guidance node not found in workflow"
    base_workflow["694"]["inputs"]["cfg"] = cfg

    # Adjust steps
    assert "751" in base_workflow, "Beta sampling scheduler node not found in workflow"
    base_workflow["751"]["inputs"]["steps"] = steps

    # === Load prompts ===
    with open(PROMPTS_PATH, "r", encoding="utf-8") as f:
        prompts = json.load(f)

    print(f"✅ Loaded {len(prompts)} prompts. Starting batch submission...")

    negative_prompt = "text, words, letters, font, watermark, signature, logo, UI, interface, button, subtitle, caption, speech bubble, comic text, manga text, handwritten, scribble, multiple subgraphs, collage, split image, diptych, triptych, panels, anime, cartoon, illustration, comic panels, photo frames, borders, grid, layout, multiple subjects, disconnected elements, disorganized, chaotic composition, floating objects, misplaced limbs, nonsensical object placement, broken anatomy, extra limbs, fused fingers, blurry, unfocused, out of focus, low resolution, bad quality, worst quality, jpeg artifacts, noise, dull colors, flat background, boring composition, nonsensical object placement"

    # === Submit each prompt ===
    for idx, item in enumerate(prompts):
        try:
            # Deep copy the workflow
            workflow = json.loads(json.dumps(base_workflow))

            # Replace prompt
            workflow[POSITIVE_NODE_ID]["inputs"]["text"] = item.get("positive", "") + " " + style_suffix
            workflow[NEGATIVE_NODE_ID]["inputs"]["text"] = negative_prompt  # item.get("negative", "")

            # Set random seed
            seed = random.randint(1, 2**48)
            workflow[SEED_NODE_ID]["inputs"]["noise_seed"] = seed

            # Set unique output prefix (to avoid overwrites)
            prefix = f"batch_{idx:04d}_seed{seed}"
            workflow[SAVE_NODE_ID]["inputs"]["filename_prefix"] = prefix

            # Construct request payload
            payload = {"prompt": workflow}

            # Submit to ComfyUI
            resp = requests.post(f"{COMFYUI_URL}/prompt", json=payload, timeout=10)
            if resp.status_code == 200:
                prompt_id = resp.json().get("prompt_id", "unknown")
                print(f"✅ [{idx+1}/{len(prompts)}] Submitted | seed={seed} | prompt_id={prompt_id}")
            else:
                print(f"❌ [{idx+1}] Submission failed: {resp.status_code} - {resp.text}")

            # Give GPU a break
            time.sleep(360)

        except Exception as e:
            print(f"💥 [{idx+1}] Script error: {e}")
            continue

    print("🏁 All tasks submitted. Please check ComfyUI/output/ directory for results.")

if __name__ == "__main__":

    # Set random seed to ensure different runs
    random.seed(time.time())

    params = [
        (4.0, 23),
        (4.5, 26),
    ]

    styles = [
        "facial lighting soft and even, professional portrait photography style, high definition, rich detail, natural skin texture.",
        "This is a high-definition, high-quality portrait photograph taken with a professional camera with well composed frame.",
        "4K resolution, rich in detail, hyper-realistic.",
        "masterpiece, best quality, photorealistic, 8K, realistic.",
    ]
    
    NUM_RUNS = 100

    for i in range(NUM_RUNS):
        cfg, steps = random.choice(params)
        style_suffix = random.choice(styles)

        print(f"\n=== Run config: cfg={cfg}, steps={steps}, style suffix='{style_suffix}', run {i+1} ===")
        func(cfg, steps, style_suffix)
        print(f"=== Completed config: cfg={cfg}, steps={steps}, style suffix='{style_suffix}', run {i+1} ===\n")
        time.sleep(60 * 2)
