# ProtonWatcher

* Written in Deno
* Automatically watch ProtonVPN log files and update qBittorrent port.
* Currently only works on Windows


## Installation
1. Download the most recent release
2. If you want the program to run for all users, move the file to 
`C:\ProgramData\Microsoft\Windows\Start Menu\Programs\StartUp`
or if you want it to run for your account only
`%appdata%\Microsoft\Windows\Start Menu\Programs\Startup`
3. Configure qBittorrent to use ProtonVPN in Preferences->Advanced->Network Interface

## Configuration

ProtonWatcher will attempt to read from a config file called `protonwatcher-config.json` in the same directory as the executable. You do not need to provide all options.

| Option     | Description                        | Default                                              |
|------------|------------------------------------|------------------------------------------------------|
| `path`     | Path to ProtonVPN log file         | `C:\\ProgramData\\ProtonVPN\\Logs\\service-logs.txt` |
| `username` | qBittorrent username               | admin                                                |
| `password` | qBittorrent password               | adminadmin                                           |
| `interval` | Interval between checking log file | 5000                                                 |
| `webPort`  | qBittorrent web UI port            | 8080                                                 |
| `baseUrl`  | Web UI protocol & hostname         | `http://localhost`                                   |