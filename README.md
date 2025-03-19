# CharCounter

CharCounter is a Go-based tool designed to analyze the frequency of characters in files with various extensions from Git repositories. This tool helps you understand the distribution of characters in code files, which can be useful for creating the best keyboard layout for programming.

## Features

- Clone a repository by URL, count the frequency of characters for all text files, and save statistics to a local database.
- Show frequency by file mask and filter from the local database.

## Usage

To use CharCounter, run the following command:

```
Usage of ./charcounter [flags] <GitRepoURL> <FileMask>
  -a    ignore letters
  -cs   case sensitive
  -n    ignore numbers
  -s    ignore symbols
  -sp   count spaces
  -top  show top N characters (default 50)
```

## Example

```
./charcounter -sp -a -top=30 https://github.com/kubernetes/kubernetes "*.go"
```


