let config = {
    "path": `C:\\ProgramData\\ProtonVPN\\Logs\\service-logs.txt`,
    "username": "admin",
    "password": "adminadmin",
    "interval": 5000, // ms
    "webPort": 8080,
    "baseUrl": "http://localhost"
}

// close command prompt



try {
    const readConfig = JSON.parse(await Deno.readTextFile("protonwatcher-config.json"))
    config = { ...config, ...readConfig }
} catch {
    console.log("No config file found, using default config")
}

console.log(config)

const ORIGIN_URL = `${config.baseUrl}:${config.webPort}`
const apiUrl = `${ORIGIN_URL}/api/v2`
const portRegex = /pair (\d{5})->\d{5}/m
let currentPort = ""

// force scan of port on startup
const stat = await Deno.stat(config.path)
let lastModifiedTime = stat.mtime!
lastModifiedTime.setMilliseconds(lastModifiedTime.getMilliseconds() - config.interval)

setInterval(async () => {
    const stat = await Deno.stat(config.path)
    const modifiedTime = stat.mtime!
    if (modifiedTime > lastModifiedTime) { // if file has been modified in the last 5 seconds
        lastModifiedTime = modifiedTime
        console.log("File modified", lastModifiedTime)
        // reverse the lines of the file
        const contents = Deno.readTextFileSync(config.path).split("\n").reverse().join("\n")

        const match = contents.match(portRegex)
        if (match) {
            if (currentPort == match[1]) { // if port is unchanged, do nothing
                return
            }
            currentPort = match[1]
            console.log(`Found a new port: ${currentPort}`)

            const response = await fetch(`${apiUrl}/auth/login`, {
                body: `username=${config.username}&password=${config.password}`,
                headers: {
                    "Content-Type": "application/x-www-form-urlencoded",
                    "Referer": ORIGIN_URL
                },
                method: "POST"
            })

            if (response.status == 403) {
                console.log("You have been rate limited!")
                return
            }
            const resp = response.headers.get("set-cookie")
            if (!resp) {
                console.log("No cookie found in headers")
                return
            }

            const cookie = resp.slice(0 + 4, 32 + 4) // slice the cookie out

            await fetch(`${apiUrl}/app/setPreferences`, {
                method: "POST",
                headers: {
                    "Set-Cookie": `SID = ${cookie}`,
                    "Content-Type": "application/x-www-form-urlencoded",
                    "Referer": ORIGIN_URL
                },
                body: `json={"listen_port": ${match[1]}}`
            })

            console.log(`Updated qBittorrent port to ${currentPort}`)
        }

    } else {
        console.log("File not modified", lastModifiedTime)
    }
}, config.interval);