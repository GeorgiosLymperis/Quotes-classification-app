import torch
from transformers import T5Tokenizer, T5ForConditionalGeneration, AutoTokenizer, AutoModelForSequenceClassification

class QuoteGenerator:
    def __init__(self, gen_model_dir: str, cls_model_dir: str, device: str | None = None,
    ):
        self.device = device or ("cuda" if torch.cuda.is_available() else "cpu")

        # Generator (T5)
        self.gen_tokenizer = T5Tokenizer.from_pretrained(gen_model_dir)
        self.gen_model = T5ForConditionalGeneration.from_pretrained(gen_model_dir)
        self.gen_model.to(self.device).eval()

        # Classifier (BERT)
        self.cls_tokenizer = AutoTokenizer.from_pretrained(cls_model_dir, use_fast=True)
        self.cls_model = AutoModelForSequenceClassification.from_pretrained(cls_model_dir)
        self.cls_model.to(self.device).eval()

        self.label2id = self.cls_model.config.label2id

    def _classify_confidence(self, text: str, topic: str) -> float:
        inputs = self.cls_tokenizer(
            text,
            return_tensors="pt",
            truncation=True,
            max_length=256,
        ).to(self.device)

        with torch.no_grad():
            logits = self.cls_model(**inputs).logits
            probs = torch.softmax(logits, dim=-1)[0]

        return float(probs[self.label2id[topic]])

    def generate(self, topic: str, max_attempts: int = 5, min_confidence: float = 0.6,
        max_length: int = 40, temperature: float = 0.8, top_p: float = 0.9, repetition_penalty: float = 1.2,
    ) -> dict:
        prompt = f"generate quote | topic: {topic}"

        best_candidate = None
        best_conf = 0.0

        for attempt in range(1, max_attempts + 1):
            inputs = self.gen_tokenizer(
                prompt,
                return_tensors="pt",
                truncation=True,
                max_length=64,
            ).to(self.device)

            with torch.no_grad():
                output = self.gen_model.generate(
                    **inputs,
                    max_length=max_length,
                    do_sample=True,
                    temperature=temperature,
                    top_p=top_p,
                    repetition_penalty=repetition_penalty,
                )

            quote = self.gen_tokenizer.decode(
                output[0], skip_special_tokens=True
                ).strip()

            conf = self._classify_confidence(quote, topic)

            if conf >= min_confidence:
                return {
                    "quote": quote,
                    "confidence": conf,
                    "accepted": True,
                    "attempts": attempt,
                }

            if conf > best_conf:
                best_candidate = quote
                best_conf = conf

        return {
            "quote": best_candidate,
            "confidence": best_conf,
            "accepted": False,
            "attempts": max_attempts,
        }
