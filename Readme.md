

## 📖 Usage

```bash
./subcheck -f <subdomains_file> [-o <output_file>] [-t <threads>] [-timeout <seconds>]
```

### Flags

| Flag | Description | Default Value |
|---|---|---|
| `-f` | Path to the text file containing subdomains (Required) | *None* |
| `-o` | Path to save live subdomains (Optional) | *None* |
| `-t` | Number of concurrent threads | `50` |
| `-timeout` | HTTP request timeout in seconds | `5` |

### Examples

**1. Basic scan:**
```bash
./subcheck -f subdomains.txt
```

**2. Save results to a file:**
```bash
./subcheck -f subdomains.txt -o live.txt
```

**3. High-speed scan with more threads:**
```bash
./subcheck -f subdomains.txt -t 100 -o live.txt
```

**4. Increase timeout for slow networks:**
```bash
./subcheck -f subdomains.txt -timeout 10
```

## 📊 Example Output

```text
[*] Found 150 unique subdomains to check
[*] Starting liveness check with 50 threads...
[+] Live: api.example.com                        | Status: 200 | URL: https://api.example.com
[-] Dead: dev.example.com                        | Error: dial tcp: lookup dev.example.com: no such host
[+] Live: admin.example.com                      | Status: 403 | URL: https://admin.example.com

[*] Scan completed. Found 45 live subdomains out of 150
[*] Results saved to: live.txt
```

## 🔒 Security Features

- **TLS Verification:** Proper SSL/TLS certificate validation (no `InsecureSkipVerify`)
- **Input Sanitization:** Automatically removes invalid characters and formats
- **Duplicate Removal:** Handles duplicate entries in input file
- **Protocol Detection:** Smart fallback from HTTPS to HTTP

## 🧪 Testing

The tool handles various scenarios:
- ✅ Valid HTTP 200 responses
- ✅ Redirect responses (301, 302)
- ✅ Client errors (403, 404)
- ✅ Server errors (500, 503)
- ✅ DNS failures
- ✅ Connection timeouts
- ✅ TLS certificate errors
- ✅ Invalid/malformed URLs

## ⚠️ Disclaimer

This tool is for educational purposes and authorized security testing only. Always obtain proper authorization before scanning any target.

## 🤝 Contributing

Contributions welcome! Open issues or submit pull requests.

## 📜 License

MIT License

