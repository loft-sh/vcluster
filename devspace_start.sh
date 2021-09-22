#!/bin/bash
set +e  # Continue on errors

COLOR_CYAN="\033[0;36m"
COLOR_RESET="\033[0m"

echo -e "${COLOR_CYAN}
   ____              ____
  |  _ \  _____   __/ ___| _ __   __ _  ___ ___
  | | | |/ _ \ \ / /\___ \| '_ \ / _\` |/ __/ _ \\
  | |_| |  __/\ V /  ___) | |_) | (_| | (_|  __/
  |____/ \___| \_/  |____/| .__/ \__,_|\___\___|
                          |_|
${COLOR_RESET}
Welcome to your development container!
This is how you can work with it:
- Run \`${COLOR_CYAN}go run -mod vendor cmd/vcluster/main.go${COLOR_RESET}\` to start vcluster
- Run \`${COLOR_CYAN}devspace enter -n vcluster --pod ${HOSTNAME} -c syncer${COLOR_RESET}\` to create another shell into this container
- Run \`${COLOR_CYAN}kubectl ...${COLOR_RESET}\` from within the container to access the vcluster if its started
- ${COLOR_CYAN}Files will be synchronized${COLOR_RESET} between your local machine and this container
"

bash