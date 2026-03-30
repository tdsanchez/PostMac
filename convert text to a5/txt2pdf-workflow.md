# Text to A5 PDF Conversion - Professional Workflow

## The Working Solution

This workflow produces professional-quality A5 PDFs with proper word wrapping and typography.

## Prerequisites

```bash
brew install pandoc basictex
```

After installation, restart your terminal or source the path:
```bash
eval "$(/usr/libexec/path_helper)"
```

## Single File Conversion

```bash
/opt/homebrew/bin/pandoc "input.txt" -o "output.pdf" \
  --pdf-engine=/Library/TeX/texbin/pdflatex \
  -V papersize=a5 \
  -V geometry:margin=15mm
```

## Batch Conversion Script

For converting all `.txt` files in a directory:

```bash
for file in *.txt; do
  /opt/homebrew/bin/pandoc "$file" -o "${file%.txt}.pdf" \
    --pdf-engine=/Library/TeX/texbin/pdflatex \
    -V papersize=a5 \
    -V geometry:margin=15mm
done
```

## Why This Works

- **Pandoc**: Handles text processing and formatting
- **LaTeX (pdflatex)**: Provides professional typesetting with proper word wrapping
- **A5 paper size**: Perfect for reading on tablets/e-readers
- **15mm margins**: Comfortable reading space

## What NOT to Use

**enscript + ps2pdf**: Breaks words arbitrarily mid-line because it uses fixed-width character positioning instead of proper word wrapping.

## Quality Results

- Professional typography
- Proper word boundaries
- Clean justified text
- Consistent formatting
- Industry-standard PDF output
