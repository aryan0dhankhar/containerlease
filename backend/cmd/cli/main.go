package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "auth":
		handleAuth(args)
	case "container":
		handleContainer(args)
	case "snapshot":
		handleSnapshot(args)
	case "admin":
		handleAdmin(args)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleAuth(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease auth <register|login|logout|who>")
		return
	}

	subCmd := args[0]
	switch subCmd {
	case "register":
		registerUser(args[1:])
	case "login":
		loginUser(args[1:])
	case "logout":
		logoutUser()
	case "who":
		whoAmI()
	default:
		fmt.Printf("unknown auth command: %s\n", subCmd)
	}
}

func handleContainer(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease container <list|provision|delete|status|logs>")
		return
	}

	subCmd := args[0]
	switch subCmd {
	case "list":
		listContainers(args[1:])
	case "provision":
		provisionContainer(args[1:])
	case "delete":
		deleteContainer(args[1:])
	case "status":
		containerStatus(args[1:])
	case "logs":
		containerLogs(args[1:])
	default:
		fmt.Printf("unknown container command: %s\n", subCmd)
	}
}

func handleSnapshot(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease snapshot <list|create|delete|restore>")
		return
	}

	subCmd := args[0]
	switch subCmd {
	case "list":
		listSnapshots(args[1:])
	case "create":
		createSnapshot(args[1:])
	case "delete":
		deleteSnapshot(args[1:])
	case "restore":
		restoreSnapshot(args[1:])
	default:
		fmt.Printf("unknown snapshot command: %s\n", subCmd)
	}
}

func handleAdmin(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease admin <users|tenants|audit>")
		return
	}

	subCmd := args[0]
	switch subCmd {
	case "users":
		listUsers(args[1:])
	case "tenants":
		listTenants(args[1:])
	case "audit":
		listAuditLog(args[1:])
	default:
		fmt.Printf("unknown admin command: %s\n", subCmd)
	}
}

// Auth commands
func registerUser(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	email := fs.String("email", "", "user email")
	username := fs.String("username", "", "username")
	password := fs.String("password", "", "password")
	tenant := fs.String("tenant", "", "tenant ID (optional)")

	fs.Parse(args)

	if *email == "" || *username == "" || *password == "" {
		fmt.Println("Error: email, username, and password are required")
		fs.PrintDefaults()
		return
	}

	payload := map[string]string{
		"email":    *email,
		"username": *username,
		"password": *password,
	}
	if *tenant != "" {
		payload["tenantId"] = *tenant
	}

	data, _ := json.Marshal(payload)
	resp, err := http.Post(getAPIURL()+"/auth/register", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if resp.StatusCode == 201 {
		fmt.Printf("✓ User registered: %s\n", *email)
		if token, ok := result["token"].(string); ok {
			saveToken(token)
		}
	} else {
		fmt.Printf("✗ Registration failed: %v\n", result)
	}
}

func loginUser(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "user email")
	password := fs.String("password", "", "password")

	fs.Parse(args)

	if *email == "" || *password == "" {
		fmt.Println("Error: email and password are required")
		fs.PrintDefaults()
		return
	}

	payload := map[string]string{"email": *email, "password": *password}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(getAPIURL()+"/auth/login", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if resp.StatusCode == 200 {
		if token, ok := result["token"].(string); ok {
			saveToken(token)
			fmt.Printf("✓ Logged in as: %s\n", *email)
		}
	} else {
		fmt.Printf("✗ Login failed: %v\n", result)
	}
}

func logoutUser() {
	os.Remove(tokenFile())
	fmt.Println("✓ Logged out")
}

func whoAmI() {
	token := loadToken()
	if token == "" {
		fmt.Println("Not logged in")
		return
	}
	fmt.Printf("✓ Logged in (token: %s...)\n", token[:20])
}

// Container commands
func listContainers(args []string) {
	_ = args
	req, _ := http.NewRequest("GET", getAPIURL()+"/containers", nil)
	addAuthHeader(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var containers []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&containers)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tCREATED")
	for _, c := range containers {
		fmt.Fprintf(w, "%v\t%v\t%v\n", c["id"], c["status"], c["createdAt"])
	}
	w.Flush()
}

func provisionContainer(args []string) {
	_ = args
	fmt.Println("Provision container: implementation pending")
}

func deleteContainer(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease container delete <container-id>")
		return
	}
	fmt.Println("Delete container: implementation pending")
}

func containerStatus(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease container status <container-id>")
		return
	}
	fmt.Println("Container status: implementation pending")
}

func containerLogs(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: containerlease container logs <container-id>")
		return
	}
	fmt.Println("Container logs: implementation pending")
}

// Snapshot commands
func listSnapshots(args []string) {
	_ = args
	fmt.Println("List snapshots: implementation pending")
}

func createSnapshot(args []string) {
	_ = args
	fmt.Println("Create snapshot: implementation pending")
}

func deleteSnapshot(args []string) {
	_ = args
	fmt.Println("Delete snapshot: implementation pending")
}

func restoreSnapshot(args []string) {
	_ = args
	fmt.Println("Restore snapshot: implementation pending")
}

// Admin commands
func listUsers(args []string) {
	_ = args
	fmt.Println("List users: admin access required")
}

func listTenants(args []string) {
	_ = args
	fmt.Println("List tenants: admin access required")
}

func listAuditLog(args []string) {
	_ = args
	fmt.Println("Audit log: admin access required")
}

// Helper functions
func getAPIURL() string {
	if url := os.Getenv("CONTAINERLEASE_API"); url != "" {
		return url
	}
	return "http://localhost:8080/api"
}

func tokenFile() string {
	home, _ := os.UserHomeDir()
	return home + "/.containerlease/token"
}

func saveToken(token string) error {
	os.MkdirAll(os.Getenv("HOME")+"/.containerlease", 0700)
	return os.WriteFile(tokenFile(), []byte(token), 0600)
}

func loadToken() string {
	data, _ := os.ReadFile(tokenFile())
	return string(data)
}

func addAuthHeader(req *http.Request) {
	token := loadToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func printUsage() {
	fmt.Print(`ContainerLease CLI

Usage:
  containerlease <command> [options]

Commands:
  auth       User authentication (register, login, logout, who)
  container  Container operations (list, provision, delete, status, logs)
  snapshot   Snapshot operations (list, create, delete, restore)
  admin      Admin operations (users, tenants, audit) - admin access required
  help       Show this help message

Environment Variables:
  CONTAINERLEASE_API    API endpoint (default: http://localhost:8080/api)

Examples:
  containerlease auth register -email user@example.com -username user -password pass
  containerlease auth login -email user@example.com -password pass
  containerlease container list
  containerlease snapshot list
`)
}
