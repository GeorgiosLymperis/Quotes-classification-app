import os

import torch
import torch.nn.functional as F
from transformers import AutoTokenizer, AutoModelForSequenceClassification

from fastapi import FastAPI
from pydantic import BaseModel
from generator import QuoteGenerator
from typing import Dict

CLF_DIR = "./models/classifier"
GENERATOR_DIR = "./models/generator"

app = FastAPI()

tokenizer = AutoTokenizer.from_pretrained(CLF_DIR, use_fast=True)
model = AutoModelForSequenceClassification.from_pretrained(CLF_DIR)
model.eval()
generator = QuoteGenerator(GENERATOR_DIR, CLF_DIR)

id2label = getattr(model.config, "id2label", None)

if id2label and isinstance(id2label, dict) and len(id2label) > 0:
    labels = [id2label[i] if i in id2label else id2label[str(i)] for i in range(len(id2label))]
else:
    labels = ["Love", "Funny", "Inspirational", "Friendship", "Life", "Motivational",
              "Sad", "Leadership", "Family", "Romantic", "Happiness", "Positive",
              "Success", "Death", "Relationship", "Humorous", "Peace", "Fashion",
              "Courage", "Money"]

class Input(BaseModel):
    text: str

class GenerateRequest(BaseModel):
    topic: str
    min_confidence: float = 0.6
    max_attempts: int = 5
    temperature: float = 0.8
    top_p: float = 0.9
    max_length: int = 40


class GenerateResponse(BaseModel):
    quote: str
    confidence: float
    accepted: bool
    attempts: int


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

@app.post("/generate", response_model=GenerateResponse)
def generate_quote(req: GenerateRequest):
    result = generator.generate(
        topic=req.topic,
        min_confidence=req.min_confidence,
        max_attempts=req.max_attempts,
        temperature=req.temperature,
        top_p=req.top_p,
        max_length=req.max_length,
    )
    return result
