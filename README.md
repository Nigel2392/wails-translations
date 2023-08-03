# The files in this folder are used with translation with a LLM

There are some issues however, which I am trying to fix / work around.

Chatgippity for example can only process so many words.

`{/* GIPPITY SPLITTER */}` is used to split up larger files, so that chatgippity can process it in multiple parts.

I have for added this to the files manually in the `docs-english (ORIGINAL)` directory, but intend to automate this in the future.

The files in this project thus may contain errors, or not work at all.

It is for now, just used to test if this is a viable concept.

To use this script, enter your private key and orginization for the chatgpt api in the main file where specified.

Then enter the following command in your CLI:

```
go run main.go <language to translate to> <dir to translate>
```
