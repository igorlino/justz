#!/bin/sh
# Runs inside a clean container (see test-completion-container.sh). It loads
# the generated completion scripts into real bash and real zsh and checks what
# they offer, then drives zsh through a pseudo-terminal like a person typing.
set -u

failures=0
pass() { echo "ok   - $1"; }
fail() { echo "FAIL - $1"; failures=$((failures + 1)); }

# check <description> <expected> <actual>
check() {
    if [ "$2" = "$3" ]; then pass "$1"; else fail "$1"; echo "       expected: $2"; echo "       actual:   $3"; fi
}

ln -sf /usr/local/bin/justz /usr/local/bin/jz
ln -sf /usr/local/bin/justz /usr/local/bin/jzh
mkdir -p /proj/.git /proj/config
cp /fixtures/commands.json /proj/config/commands.json
printf 'default:\n' > /proj/justfile
cd /proj || exit 1

# --- bash: call the completion function the way bash does -----------------
# bash_offers <description> <expected values, space separated> <command words...>
bash_offers() {
    desc="$1"; want="$2"; shift 2
    got="$(bash -c 'eval "$(justz completions bash)"
        COMP_WORDS=("$@"); COMP_CWORD=$(($# - 1)); _justz_complete
        printf "%s " "${COMPREPLY[@]}"' _ "$@")"
    check "bash: $desc" "$want " "$got"
}
bash_offers "jz <TAB> lists core commands then groups" "build ci test build deploy tool" jz ""
bash_offers "jz tool <TAB> lists the group's commands" "clean doctor" jz tool ""
bash_offers "jz build docker <TAB> lists the options" "--push" jz build docker ""
bash_offers "an enum flag offers its values in the author's order" "low medium high" jz release --level ""
bash_offers "jzh dep<TAB> completes a group" "deploy" jzh dep
bash_offers "jzh -<TAB> offers its flags" "-b -j -w" jzh -
bash_offers "justz ch<TAB> completes a command" "check" justz ch
bash_offers "justz shim <TAB> lists the subcommands" "add doctor list remove" justz shim ""
bash_offers "justz completions <TAB> lists the shells" "bash zsh" justz completions ""
files="$(bash -c 'eval "$(justz completions bash)"; COMP_WORDS=(jz test -k ""); COMP_CWORD=3; _justz_complete; printf "%s " "${COMPREPLY[@]}"' | tr ' ' '\n' | grep -c 'justfile')"
check "bash: a free-text flag falls back to file names" "1" "$files"

# --- zsh: call the function with the compsys helpers stubbed --------------
# zsh_offers <description> <expected values> <command words...>
zsh_offers() {
    desc="$1"; want="$2"; shift 2
    got="$(zsh -f -c '
        _describe() { shift 2; local arr=$2; local -a v; v=(${${(P)arr}%%:*}); print -rn -- "${(j: :)v}" }
        _files() { print -rn -- "<files>" }
        eval "$(justz completions zsh)"
        words=("$@"); CURRENT=$#; _justz_complete' _ "$@")"
    check "zsh: $desc" "$want" "$got"
}
zsh_offers "jz <TAB> lists core commands then groups" "build ci test build deploy tool" jz ""
zsh_offers "jz tool <TAB> lists the group's commands" "clean doctor" jz tool ""
zsh_offers "jz build docker <TAB> lists the options" "--push" jz build docker ""
zsh_offers "an enum flag offers its values in the author's order" "low medium high" jz release --level ""
zsh_offers "a free-text flag falls back to file names" "<files>" jz test -k ""
zsh_offers "jzh dep<TAB> completes a group" "deploy" jzh dep
zsh_offers "justz fmt <TAB> offers files" "<files>" justz fmt ""
zsh_offers "justz shim <TAB> lists the subcommands" "add doctor list remove" justz shim ""

# --- zsh through a pseudo-terminal: type, press TAB, read the screen --------
screen="$(zsh -f -c '
    zmodload zsh/zpty
    zpty zt "zsh -f -i"
    zpty -w zt "PS1=\"%% \"; autoload -U compinit; compinit -u -d /tmp/zcompdump; eval \"\$(justz completions zsh)\""
    sleep 1.5
    zpty -r -t zt junk "*"
    zpty -w -n zt $'"'"'jz dep\t'"'"'
    sleep 1.2
    buf=""
    while zpty -r -t zt chunk "*"; do buf+=$chunk; done
    zpty -d zt
    print -r -- "$buf"' 2>&1 | sed 's/\x1b\[[0-9;?]*[A-Za-z]//g' | tr -d '\r')"
case "$screen" in
    *deploy*) pass "zsh pty: typing 'jz dep' and pressing TAB completes to deploy" ;;
    *) fail "zsh pty: typing 'jz dep' and pressing TAB completes to deploy"; echo "$screen" | tail -5 | sed 's/^/       /' ;;
esac

if [ "$failures" -ne 0 ]; then
    echo "$failures check(s) failed"
    exit 1
fi
echo "all completion checks passed"
