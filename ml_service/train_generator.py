import numpy as np # linear algebra
import pandas as pd # data processing, CSV file I/O (e.g. pd.read_csv)
import os
import json
import random
from dataclasses import dataclass
from typing import Dict, List

import numpy as np
import pandas as pd
import torch
from sklearn.model_selection import train_test_split

from datasets import Dataset
from transformers import (
    T5Tokenizer,
    T5ForConditionalGeneration,
    Trainer,
    TrainingArguments,
    DataCollatorForSeq2Seq,
    EarlyStoppingCallback,
)

SEED = 42
random.seed(SEED)
np.random.seed(SEED)
torch.manual_seed(SEED)

MODEL_NAME = "t5-small"
DATA_PATH = "quotes_dataset.csv"
OUTPUT_DIR = "models/generator"

MAX_INPUT_LEN = 64
MAX_TARGET_LEN = 48

TRAIN_RATIO = 0.50
VAL_RATIO = 0.50

df = pd.read_csv(DATA_PATH)
df = df.dropna(subset=["quote", "topic"])
df["quote"] = df["quote"].str.strip()
df["topic"] = df["topic"].str.strip()
df = df.sample(frac=1.0, random_state=SEED).reset_index(drop=True)

def build_prompt(topic: str) -> str:
    return f"generate quote | topic: {topic}"

df["input_text"] = df["topic"].apply(build_prompt)
df["target_text"] = df["quote"]

train_df, val_df = train_test_split(df, train_size=TRAIN_RATIO, random_state=SEED, shuffle=True, stratify=df["topic"])
val_df, test_df = train_test_split(val_df, train_size=VAL_RATIO, random_state=SEED, shuffle=True, stratify=val_df["topic"])

train_ds = Dataset.from_pandas(train_df[["input_text", "target_text"]])
val_ds = Dataset.from_pandas(val_df[["input_text", "target_text"]])
test_ds = Dataset.from_pandas(test_df[["input_text", "target_text"]])

tokenizer = T5Tokenizer.from_pretrained(MODEL_NAME)
model = T5ForConditionalGeneration.from_pretrained(MODEL_NAME)

def tokenize(batch: Dict[str, List[str]]) -> Dict[str, torch.Tensor]:
    inputs = tokenizer(
        batch["input_text"],
        padding="max_length",
        truncation=True,
        max_length=MAX_INPUT_LEN,
        return_tensors="pt",
        )

    with tokenizer.as_target_tokenizer():
        targets = tokenizer(
            batch["target_text"],
            padding="max_length",
            truncation=True,
            max_length=MAX_TARGET_LEN,
            return_tensors="pt",
        )

    inputs["labels"] = targets["input_ids"]
    return inputs

train_tok = train_ds.map(tokenize, batched=True, remove_columns=train_ds.column_names)
val_tok = val_ds.map(tokenize, batched=True, remove_columns=val_ds.column_names)

train_tok.set_format("torch")
val_tok.set_format("torch")

data_collator = DataCollatorForSeq2Seq(
    tokenizer=tokenizer,
    model=model,
)

training_args = TrainingArguments(
    output_dir=OUTPUT_DIR,
    eval_strategy="epoch",
    save_strategy="epoch",
    learning_rate=3e-5,
    per_device_train_batch_size=8,
    per_device_eval_batch_size=8,
    gradient_accumulation_steps=2,
    num_train_epochs=12,
    weight_decay=0.01,
    logging_dir=f"{OUTPUT_DIR}/logs",
    logging_steps=50,
    save_total_limit=2,
    load_best_model_at_end=True,
    metric_for_best_model="eval_loss",
    greater_is_better=False,
    report_to="none",
    fp16=torch.cuda.is_available(),
)

trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=train_tok,
    eval_dataset=val_tok,
    processing_class=tokenizer,
    data_collator=data_collator,
    callbacks=[
        EarlyStoppingCallback(
            early_stopping_patience=2,
            early_stopping_threshold=0.001
        )
    ],
)

trainer.train()

trainer.save_model(OUTPUT_DIR)
tokenizer.save_pretrained(OUTPUT_DIR)

# Save metadata
meta = {
    "model": MODEL_NAME,
    "task": "topic-conditioned quote generation",
    "max_input_len": MAX_INPUT_LEN,
    "max_target_len": MAX_TARGET_LEN,
    "seed": SEED,
}
with open(os.path.join(OUTPUT_DIR, "meta.json"), "w") as f:
    json.dump(meta, f, indent=2)

print(f"Model saved to {OUTPUT_DIR}")