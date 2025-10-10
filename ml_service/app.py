import os

import torch
import torch.nn.functional as F
from transformers import AutoTokenizer, AutoModelForSequenceClassification

from fastapi import FastAPI
from pydantic import BaseModel
from typing import Dict

MODEL_DIR = os.getenv("MODEL_DIR", "./model")

app = FastAPI()

tokenizer = AutoTokenizer.from_pretrained(MODEL_DIR, use_fast=True)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_DIR)
model.eval()

id2label = getattr(model.config, "id2label", None)
print(id2label)
if id2label and isinstance(id2label, dict) and len(id2label) > 0:
    labels = [id2label[i] if i in id2label else id2label[str(i)] for i in range(len(id2label))]
else:
    labels = ["Love", "Funny", "Inspirational", "Friendship", "Life", "Motivational",
              "Sad", "Leadership", "Family", "Romantic", "Happiness", "Positive",
              "Success", "Death", "Relationship", "Humorous", "Peace", "Fashion",
              "Courage", "Money"]

class Input(BaseModel):
    text: str

@app.get("/health")
def health():
    return {"status": "ok", "model_loaded": True, "num_labels": len(labels)}

@app.post("/classify")
def classify(user_input: Input) -> Dict:
    tokenized_inputs = tokenizer(user_input.text, padding=True, truncation=True, max_length=256, return_tensors="pt")
    with torch.no_grad():
        logits = model(**tokenized_inputs).logits
    probs = torch.softmax(logits, dim=-1).cpu().numpy()[0]  # shape: [num_labels]
    top3_indices = probs.argsort()[-3:][::-1]  # top 3 in descending order
    top, second, third = top3_indices
    return {
        "label": id2label[top],
        "probs": {
            id2label[top]: round(float(probs[top]), 4),
            id2label[second]: round(float(probs[second]), 4),
            id2label[third]: round(float(probs[third]), 4)
        },
        "model_version": os.getenv("MODEL_VERSION", "v1")
    }
