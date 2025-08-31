# ValCortex + Gemma-2 Setup (Windows)

## -0.1 โฟลเดอร์ปลายทาง
```powershell
New-Item -ItemType Directory -Force -Path G:\models | Out-Null
```

# Gemma-2 2B Instruct (GGUF, Q4_K_M)
```powershell
curl.exe -L "https://huggingface.co/bartowski/gemma-2-2b-it-GGUF/resolve/main/gemma-2-2b-it-Q4_K_M.gguf?download=true" -o "G:\models\gemma-2-2b-it-Q4_K_M.gguf"
```


# ValCortex + vicuna-13b Setup (Windows)

## 0. ดาวน์โหลดโมเดล (.gguf)

แนะนำเริ่มด้วย quant Q4_K_M (สมดุลคุณภาพ/แรม)

# โฟลเดอร์ปลายทาง
```powershell
New-Item -ItemType Directory -Force -Path G:\models | Out-Null
```

# ดาวน์โหลดไฟล์เดียว (Q4_K_M ~7.9GB)
```powershell
curl.exe -L "https://huggingface.co/cjpais/llava-v1.6-vicuna-13b-gguf/resolve/main/llava-v1.6-vicuna-13b.Q4_K_M.gguf?download=true" -o "G:\models\llava-v1.6-vicuna-13b.Q4_K_M.gguf"
```


# ValCortex + GPT-OSS-20B Setup (Windows)

## 1. ดาวน์โหลดโมเดล (.gguf)

> โมเดลจะถูกเก็บไว้ที่ `G:\models`

```powershell
# สร้างโฟลเดอร์เก็บโมเดล (ถ้ายังไม่มี)
New-Item -ItemType Directory -Force -Path G:\models | Out-Null
```

```powershell
# ดาวน์โหลดไฟล์ quant แบบ Q4_K_M (ประหยัดแรม ~16GB+)
curl.exe -L "https://huggingface.co/unsloth/gpt-oss-20b-GGUF/resolve/main/gpt-oss-20b-Q4_K_M.gguf?download=true" -o "G:\models\gpt-oss-20b-Q4_K_M.gguf"
```

📌 ที่มา Hugging Face: [unsloth/gpt-oss-20b-GGUF](https://huggingface.co/unsloth/gpt-oss-20b-GGUF)  
(เลือก quant อื่นๆ ได้ เช่น Q5_K_M, Q8_0 ตามสเปกเครื่อง)

---




Gemma2B Single-EXE Capsule
==========================
- แค่รัน EXE ก็พอ: โปรแกรมจะดาวน์โหลด Ollama เอง (zip), แตกไปที่ .\bin\, แล้วสร้าง Modelfile อัตโนมัติ
- Modelfile ตั้งต้นชี้ไปที่ G:\models\gemma-2-2b-it-Q4_K_M.gguf (แก้ไฟล์ Modelfile ได้ถ้าพาธคุณต่าง)
- เปิด REST API: http://127.0.0.1:8088  (POST /chat)

วิธีใช้
1) แตก zip ไปที่ G:\ai\gemma2b_singleexe\
2) PowerShell:
   cd G:\ai\gemma2b_singleexe
   .\build.ps1
   .\gemma2b.exe
3) ทดสอบ:
   curl -X POST "http://127.0.0.1:8088/chat" -H "Content-Type: application/json" -d "{`"prompt`":`"สวัสดี แนะนำตัว 3 บรรทัด`"}"



curl.exe -s "http://127.0.0.1:8088/models"

curl.exe -s -X POST "http://127.0.0.1:8088/select" -H "Content-Type: application/json" -d "{\"tag\":\"gguf-gpt-oss-20b-q4_k_m\"}"

curl.exe -s -X POST "http://127.0.0.1:8088/chat" -H "Content-Type: application/json" -d "{\"prompt\":\"แนะนำตัวเอง 3 บรรทัด\"}"

curl.exe -s -X POST "http://127.0.0.1:8088/chat" -H "Content-Type: application/json" -d "{\"model\":\"gguf-gemma-2-2b-it-q4_k_m\",\"prompt\":\"อธิบายความสามารถของคุณ\"}"

curl.exe -s -X POST "http://127.0.0.1:8088/chat" -H "Content-Type: application/json" -d "{\"model\":\"gguf-llava-v1-6-vicuna-13b-q4_k_m\",\"prompt\":\"สรุปสั้น ๆ เกี่ยวกับตัวเอง\"}"

curl.exe -s -X POST "http://127.0.0.1:11434/api/chat" -H "Content-Type: application/json" -d "{\"model\":\"gguf-llava-v1-6-vicuna-13b-q4_k_m\",\"messages\":[{\"role\":\"user\",\"content\":\"บรรยายภาพนี้\",\"images\":[\"BASE64_IMAGE_HERE\"]}]}"
