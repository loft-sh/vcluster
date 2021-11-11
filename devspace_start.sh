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

If you wish to run vcluster in the debug mode with delve, run:
  \`${COLOR_CYAN}dlv debug ./cmd/vcluster/main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags=\"-mod=vendor\" -- --lease-duration=99999${COLOR_RESET}\`
  Wait until the \`${COLOR_CYAN}API server listening at: [::]:2345${COLOR_RESET}\` message appears
  Start the \"Debug vcluster (localhost:2346)\" configuration in VSCode to connect your debugger session.
  ${COLOR_CYAN}Note:${COLOR_RESET} vcluster won't start you connect with the debugger.
  ${COLOR_CYAN}Note:${COLOR_RESET} vcluster will be stopped once you detach your debugger session.
"

bash