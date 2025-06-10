#!/bin/bash
set +e  # Continue on errors

COLOR_CYAN="\033[0;36m"
COLOR_RESET="\033[0m"

RUN_CMD="go run -mod vendor cmd/vcluster/main.go start"
DEBUG_CMD="dlv debug ./cmd/vcluster/main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags=\"-mod=vendor\" -- start"

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
- Run \`${COLOR_CYAN}${RUN_CMD}${COLOR_RESET}\` to start vcluster
- Run \`${COLOR_CYAN}devspace enter -n vcluster --pod ${POD_NAME} -c syncer${COLOR_RESET}\` to create another shell into this container
- Run \`${COLOR_CYAN}kubectl ...${COLOR_RESET}\` from within the container to access the vcluster if its started
- ${COLOR_CYAN}Files will be synchronized${COLOR_RESET} between your local machine and this container

${COLOR_CYAN}NOTE:${COLOR_RESET} you may need to provide additional flags through the command line, because the flags set from the chart are ignored in the dev mode.

If you wish to run vcluster in the debug mode with delve, run:
  \`${COLOR_CYAN}${DEBUG_CMD}${COLOR_RESET}\`
  Wait until the \`${COLOR_CYAN}API server listening at: [::]:2345${COLOR_RESET}\` message appears
  Start the \"Debug vcluster (localhost:2346)\" configuration in VSCode to connect your debugger session.
  ${COLOR_CYAN}Note:${COLOR_RESET} vcluster won't start until you connect with the debugger.
  ${COLOR_CYAN}Note:${COLOR_RESET} vcluster will be stopped once you detach your debugger session.

${COLOR_CYAN}TIP:${COLOR_RESET} hit an up arrow on your keyboard to find the commands mentioned above :)
"
# add useful commands to the history for convenience
export HISTFILE=/tmp/.bash_history
history -s $DEBUG_CMD
history -s $RUN_CMD
history -a

# hide "I have no name!" from the bash prompt when running as non root
bash --init-file <(echo "export PS1=\"\\H:\\W\\$ \"")
