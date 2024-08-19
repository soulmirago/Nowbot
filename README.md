# Nowbot
Nowbot

HOW TO INSTALL THE BOT
1. Download and build the code

    go install github.com/USER/Nowbot/cmd/Nowbot@latest

3. Create systemd service file to run the binary as a service

    See: https://medium.com/@david.franko1998/automating-linux-scripts-with-systemd-a-quickstart-guide-b00a143c31e5

    Create the service file, replacing [username] with actual user name
   
        sudo nano  /etc/systemd/system/Nowbot.service

        Insert following content:
        [Unit]
        Description=Nowbot Discord Bot
        Wants=network-online.target
        After=network.target

        [Service]
        ExecStart=/home/[user]/go/bin/Nowbot -t "Bot [TOKEN]" -o [OWNER_ID]
        User=[username]
        Restart=on-failure
        RestartSec=10s
        StartLimitBurst=5
        StandardOutput=console

        [Install]
        WantedBy=multi-user.target

4. Enable the service

    sudo systemctl daemon-reload
    sudo systemctl enable Nowbot.service
    sudo systemctl start Nowbot.service

5. Monitor status of the service or see logs

    systemctl status Nowbot.service
    journalctl -r -u Nowbot.service


6. HOW TO RUN THE BOT MANUALLY IF NOT USING SYSTEMD

    Nowbot -t "Bot [TOKEN]" -o [OWNER_ID]

