import json
import subprocess
import time

def patch_node():
    node_name = "blender"
    # Get current node status
    cmd = f"kubectl get node {node_name} -o json"
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"Error getting node: {result.stderr}")
        return

    node = json.loads(result.stdout)
    
    # Prepare the patch
    patch = {
        "status": {
            "capacity": {
                "nvidia.com/gpu": "1"
            },
            "allocatable": {
                "nvidia.com/gpu": "1"
            }
        }
    }
    
    # Save patch to file
    with open("node_patch.json", "w") as f:
        json.dump(patch, f)
        
    # Use kubectl proxy to patch status (since patch status is tricky via CLI)
    proxy = subprocess.Popen("kubectl proxy --port=8001", shell=True)
    time.sleep(2)
    
    try:
        cmd = f'curl -X PATCH -H "Content-Type: application/merge-patch+json" http://localhost:8001/api/v1/nodes/{node_name}/status --data @node_patch.json'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        print(f"Patch result: {result.stdout}")
    finally:
        proxy.terminate()

if __name__ == "__main__":
    patch_node()
