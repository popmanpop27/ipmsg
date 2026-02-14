# ipmsg
A simple program for chatting between computers on a local network.

ipmsg is a small Go tool that listens on port `6767` (-_(*_*)_-) and receives TCP requests, writing them to a file `ipmsg.txt` located in your home directory. It also includes a client that can send messages to a specific IP in your network or broadcast to all devices on the local network.

That is how chat looks like:

```
TIME                 | FROM                           |    LEN
---------------------------------------------------------------
2026-02-05 21:23:20  | 192.168.1.39                   |     17
hello


2026-02-06 21:17:57  | 192.168.1.39                   |      6

```

## Table of Contents
1. [Installation](#installation)
2. [Usage](#usage)
3. [Features](#features)
4. [Configuration](#configuration)
5. [Contributing](#contributing)
6. [License](#license)

## Installation

1. Clone the repository:
    ```bash
    git clone https://github.com/popmanpop27/ipmsg.git
    cd ipmsg
    ```

2. Run the installation script for your system (macOS, Linux, or Windows):
    ```bash
    # Linux / macOS
    chmod +x linux_install.sh
    ./linux_install.sh

    # Windows
    Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
    .\win_install.ps1
    ```

## Usage

1. Send a message to all computers on your local network:
    ```bash
    ipmsg
    ```

2. Send a message to a specific device using the `--to` flag:
    ```bash
    ipmsg --to 192.168.1.1
    ```

3. Change host, port, or message file by using the server binary flags. To see all options, run:
    ```bash
    ipmsg --help
    ```

All received messages will be written to file ipmsg.txt located in your home directory
```
    // message getting
    //linux/mac

    cat ~/ipmsg.txt

    //windows

    type C:\Users\{User}\ipmsg.txt
```

Also you can name addresses with that command

```ipmsg --alias <name_for_address> --ip <address>```

After what you can send messages to that alias and it will be interpreted as ip

```ipmsg --to alex```

And in ~/ipmsg.txt file all messages will be marked as from alex

```
2026-02-08 16:17:39  | alex(172.168.1.39)             |     20
to alan from alex
```

To prevent everyone from creating aliases, with each of your messages you send your name, so all computers are named. 
If you don't want to name your IP, press Enter when asked for your name when sending

## Features
- Simple local network chat
- Named devices in net
- Logs messages to a file in your home directory (`ipmsg.txt`)
- Send messages to a specific IP or broadcast to all devices
- Easy installation scripts for Linux, macOS, and Windows

## Configuration
- Default port: `6767`
- Default message file: `~/ipmsg.txt`
- Change these settings using server flags.

## Contributing
1. Fork the repository
2. Create a new branch (`git checkout -b feature-name`)
3. Make your changes
4. Submit a pull request

## License
This project is licensed under the MIT License.
