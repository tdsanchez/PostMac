git grep -ohI '\b[a-zA-Z]\+\b' | tr '[:upper:]' '[:lower:]' | sort | uniq -c | sort -rn 
