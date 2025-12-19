#!/bin/bash
# Convert all .txt files to A5 PDFs using pandoc

PANDOC="/opt/homebrew/bin/pandoc"
PDFLATEX="/Library/TeX/texbin/pdflatex"

total=0
converted=0
skipped=0

# Find all .txt files
while IFS= read -r -d '' txtfile; do
    ((total++))

    # Generate PDF filename
    pdffile="${txtfile%.txt}.pdf"

    # Skip if PDF already exists
    if [ -f "$pdffile" ]; then
        ((skipped++))
        echo "[$total] SKIP: $(basename "$txtfile") (PDF exists)"
        continue
    fi

    # Convert
    echo "[$total] Converting: $(basename "$txtfile")"
    if "$PANDOC" "$txtfile" -o "$pdffile" \
        --pdf-engine="$PDFLATEX" \
        -V papersize=a5 \
        -V geometry:margin=15mm 2>/dev/null; then
        ((converted++))
        echo "[$total] ✓ Created: $(basename "$pdffile")"
    else
        echo "[$total] ✗ FAILED: $(basename "$txtfile")"
    fi

done < <(find /Volumes/Publica/books -name "*.txt" -type f -print0)

echo ""
echo "========================================="
echo "Conversion complete!"
echo "Total files: $total"
echo "Converted: $converted"
echo "Skipped (already exist): $skipped"
echo "========================================="
