data "tofukit_query" "ls" {
  instructions = [
    "What color is sky?",
    "Answer with exactly one word, in lowercase",
    "Do not include any punctuation or additional text",
  ]

  format = "text"
}
