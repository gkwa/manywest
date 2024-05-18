#!/usr/bin/env bash
set -e

tmp=$(mktemp -d {{.Cwd}}.XXXXX)

if [ -z "${tmp+x}" ] || [ -z "$tmp" ]; then
    echo "error: $tmp is not set or is an empty string."
    exit 1
fi

if ! command -v txtar-c >/dev/null; then
    echo go install github.com/rogpeppe/go-internal/cmd/txtar-c@latest
	exit 1
fi

declare -a files=(
	{{range .Files}}# {{.Path}} # loc: {{.Count}}
	{{end}}
)
for file in "${files[@]}"; do
    echo $file
done | tee $tmp/filelist.txt

tar -cf $tmp/{{.Cwd}}.tar -T $tmp/filelist.txt
mkdir -p $tmp/{{.Cwd}}
tar xf $tmp/{{.Cwd}}.tar -C $tmp/{{.Cwd}}
rg --hidden --files $tmp/{{.Cwd}}

mkdir -p $tmp/gpt_instructions_XXYYBB

cat >$tmp/gpt_instructions_XXYYBB/1.txt <<EOF
{{ if .IncludeInstructions }}
Subject: Code Submission Guidelines in Txtar Archive Format

As we collaborate on code submissions, I would like to emphasize some
guidelines for presenting your code using the txtar archive format.

Unified Code Block: Ensure that all your code is displayed within a
single code block using the txtar archive format.

This helps maintain a structured and organized presentation.

Modification Verification: If, upon review, you find that you haven't
made any modifications to a specific source file since its initial
state, kindly refrain from including it in the code block.

Txtar Archive Format Summary: The txtar archive format should follow
this structure:

#+begin_example
-- cmd/main.go --
{ contents of main.go }
-- mypackage.go --
{ contents of mypackage.go }
#+end_example

Omitting Unchanged Files:
If a file requires no changes, please exclude it from the txtar archive.

Do not include statements like this example:
#+begin_example
// ... (unchanged) or similar indications.
#+end_example

Avoid Partial Listings:
Refrain from providing partial listings for unchanged files.

Instead, either omit the file entirely or include its complete content
without any abbreviations or explanations about unchanged portions.

Your adherence to these guidelines will greatly facilitate our
collaboration and ensure a streamlined code submission process.

Thank you for your attention to detail and cooperation.
{{ end }}
EOF

{
    cat $tmp/gpt_instructions_XXYYBB/1.txt
    echo txtar archive is below
    txtar-c -quote -a $tmp/{{.Cwd}}
} | pbcopy

rm -rf $tmp
