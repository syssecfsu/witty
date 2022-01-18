echo "Try to resize shell with shell command"
printf '\e[8;36;120t'
clear
$(dirname $0)/replay "$@"