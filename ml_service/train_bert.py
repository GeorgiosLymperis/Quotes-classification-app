import json

import numpy as np
import pandas as pd

import matplotlib.pyplot as plt
import seaborn as sns
import squarify

from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score, classification_report, confusion_matrix, f1_score
from imblearn.over_sampling import RandomOverSampler


import torch
from datasets import Dataset
from transformers import (
    Trainer, TrainingArguments, EarlyStoppingCallback, pipeline,
    AutoTokenizer, AutoModelForSequenceClassification
)

np.random.seed(42)
torch.manual_seed(42)

data = pd.read_csv("dataset.csv")
data = data.drop_duplicates(subset='quote', keep='first')

# Split data: small training set due to limited resources
train_df, temp_df = train_test_split(data, test_size=0.8, random_state=42, stratify=data['topic'])

# Split remaining data into validation and test sets
val_df, test_df = train_test_split(temp_df, test_size=0.3, random_state=42, stratify=temp_df['topic'])

print(f"Train size: {len(train_df)}")
print(f"Validation size: {len(val_df)}")
print(f"Test size: {len(test_df)}")

TOPICS = [
    "Love", "Funny", "Inspirational", "Friendship", "Life", "Motivational",
    "Sad", "Leadership", "Family", "Romantic", "Happiness", "Positive",
    "Success", "Death", "Relationship", "Bible", "Peace", "Fashion",
    "Courage", "Money"
]

# Create label mappings
label2id = {lbl: i for i, lbl in enumerate(TOPICS)}
id2label = {i: lbl for lbl, i in label2id.items()}

# Apply label encoding
train_df["label"] = train_df['topic'].map(label2id)
val_df["label"] = val_df['topic'].map(label2id)
test_df["label"] = test_df['topic'].map(label2id)

# Separate features and labels
X = train_df[['quote']]
y = train_df[['label']]

# Apply random oversampling
sampler = RandomOverSampler(random_state=42)
X_resampled, y_resampled = sampler.fit_resample(X, y)

train_df_resampled = pd.DataFrame({
    "text": X_resampled["quote"].values,
    "label": y_resampled.values.reshape(-1)
})

val_df_fixed = pd.DataFrame({
    "text": val_df["quote"].tolist(),
    "label": val_df["label"].tolist()
})
test_df_fixed = pd.DataFrame({
    "text": test_df["quote"].tolist(),
    "label": test_df["label"].tolist()
})

print(f"Validation size: {len(val_df_fixed)}")
print(f"Test size: {len(test_df_fixed)}")

train_dataset = Dataset.from_pandas(train_df_resampled)  # columns: text, label
val_dataset = Dataset.from_pandas(val_df_fixed)
test_dataset = Dataset.from_pandas(test_df_fixed)

# Define pretrained model checkpoint
model_name = "bert-base-uncased"

# Load tokenizer
tokenizer = AutoTokenizer.from_pretrained(model_name, use_fast=True)

# Load model for sequence classification
model = AutoModelForSequenceClassification.from_pretrained(
    model_name,
    num_labels=len(label2id),
    id2label=id2label,
    label2id=label2id
)

def tokenize_function(examples):
    return tokenizer(
        examples['text'],              # column containing text
        padding="max_length",          # pad all sequences to the same length
        truncation=True,               # truncate longer sequences
        max_length=512                 # maximum token length for BERT
    )

train_tokenized = train_dataset.map(tokenize_function, batched=True)
val_tokenized = val_dataset.map(tokenize_function, batched=True)
test_tokenized = test_dataset.map(tokenize_function, batched=True)

# HF expects the target column to be named "labels"
train_tokenized = train_tokenized.rename_column("label", "labels")
val_tokenized = val_tokenized.rename_column("label", "labels")
test_tokenized = test_tokenized.rename_column("label", "labels")

# set tensor format
train_tokenized.set_format("torch")
val_tokenized.set_format("torch")
test_tokenized.set_format("torch")

# Define training arguments for fine-tuning the BERT model
training_args = TrainingArguments(
    output_dir="./model",                  # Directory to store model checkpoints and outputs
    eval_strategy="epoch",                 # Evaluate the model at the end of each epoch
    learning_rate=2e-5,                    # Learning rate for AdamW optimizer
    per_device_train_batch_size=8,         # Batch size per device
    num_train_epochs=10,                   # Total number of training epochs
    weight_decay=0.01,                     # Weight decay for regularization
    logging_dir="./logs",                  # Directory to save training logs
    logging_steps=10,                      # Log metrics every 10 steps
    save_strategy="epoch",                 # Save model checkpoint at the end of each epoch
    load_best_model_at_end=True,           # Load the best model based on eval metric after training
    metric_for_best_model="f1_macro",      # Metric to determine the best model
    greater_is_better=True,                # Higher F1 score is better
    save_total_limit=2,                    # Keep only the last 2 checkpoints
    run_name="bert_quote_classification"   # Name for experiment tracking/logging
)

# Define early stopping callback — stop training when performance stops improving
early_stopping = EarlyStoppingCallback(
    early_stopping_patience=2,             # Number of epochs without improvement before stopping
    early_stopping_threshold=0.001         # Minimum improvement threshold to count as progress
)

# Define metric computation function for validation
def compute_metrics(eval_pred):
    logits, labels = eval_pred
    preds = np.argmax(logits, axis=-1)
    return {
        "accuracy": accuracy_score(labels, preds),
        "f1_macro": f1_score(labels, preds, average="macro"),
        "f1_weighted": f1_score(labels, preds, average="weighted"),
    }

# Initialize the Trainer — handles training, evaluation, and checkpointing
trainer = Trainer(
    model=model,                           # The model to be fine-tuned
    args=training_args,                    # Training configuration defined above
    train_dataset=train_tokenized,         # Tokenized training dataset
    eval_dataset=val_tokenized,            # Tokenized validation dataset
    compute_metrics=compute_metrics,       # Evaluation metrics function
    callbacks=[early_stopping]             # Early stopping callback
)

trainer.train()

trainer.save_model("./models/classifier")
tokenizer.save_pretrained("./models/classifier")

with open("./models/classifier/label_mapping.json", "w") as f:
    json.dump({"label2id": label2id, "id2label": id2label}, f, indent=2)