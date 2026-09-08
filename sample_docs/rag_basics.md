# Retrieval-Augmented Generation Basics

Retrieval-augmented generation, or RAG, combines a search step with a
generation step. Instead of asking a language model to answer purely
from what it memorized during training, the system first retrieves
relevant passages from an external corpus and includes them in the
prompt, letting the model ground its answer in specific source text.

The retrieval step usually relies on vector embeddings: a piece of text
is converted into a fixed-length numeric vector such that semantically
similar text produces vectors that are close together, typically
measured with cosine similarity. To answer a question, the question
itself is embedded and compared against every stored chunk's vector,
and the closest matches are pulled out as context.

Chunking matters because embedding an entire document as one vector
loses fine-grained detail, while chunking too aggressively can cut a
sentence in half and lose meaning. A common approach is to chunk by a
fixed word or token count with some overlap between consecutive chunks,
so an idea that straddles a boundary still appears intact in at least
one chunk.

RAG systems are attractive because they update as the underlying corpus
changes without retraining the model, and because responses can cite
their sources, which makes them easier to trust and to debug than a
model answering from memory alone.
