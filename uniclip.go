package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	// "io/ioutil"
	// "bytes"
	// "compress/flate"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/term"
)

var (
	secondsBetweenChecksForClipChange = 1
	listOfClients                     = make([]*bufio.Writer, 0)
	localClipboard                    string
	cryptoStrength                    = 16384
	password                          []byte

	version        = "2.3.6"
	printVersion   bool
	verboseLogging bool
	pullBased      bool
	copyOnPaste    bool
	encryption     bool
	jsonOutput     bool
	targetAddress  string
	listenAddress  string
	serverPort     int
	pullInterval   int
	logger         *slog.Logger
)

func init() {
	flag.BoolVar(&printVersion, "version", false, "Prints the installed version")
	flag.BoolVar(&verboseLogging, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&pullBased, "pull", false, "Change the Push-Based clipboard to a Pull-Based one")
	flag.BoolVar(&copyOnPaste, "copy", false, "Change the Push-Based clipboard to a Copy-On-Paste one")
	flag.BoolVar(&encryption, "encrypt", true, "Enable encryption on all connections")
	flag.BoolVar(&jsonOutput, "json", false, "Enable json output for logging when verbose logging is enabled")
	flag.StringVar(&targetAddress, "target", "", "The address of the clipboard server to join")
	flag.StringVar(&listenAddress, "listen", "0.0.0.0", "Listen address excluding the port")
	flag.IntVar(&serverPort, "port", 38551, "server port")
	flag.IntVar(&pullInterval, "interval", 1, "Pull interval in seconds if Pull-Based clipboard is enabled")
}

func main() {
	flag.Parse()

	if printVersion {
		fmt.Printf("uniclip %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	logger = slog.New(slog.DiscardHandler)
	if verboseLogging {
		logger = slog.Default()
		slog.SetLogLoggerLevel(slog.LevelDebug)

		if jsonOutput {
			logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	}

	if targetAddress != "" {
		ConnectToServer(targetAddress)
		return
	}

	if serverPort > 65535 || serverPort < 1 {
		fmt.Println("Invalid port number:", serverPort)
		return
	}

	if encryption {
		fmt.Print("Password for -encrypt: ")
		var err error

		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println("Error reading password:", err)
			return
		}

		fmt.Println()
	}

	makeServer()
}

func makeServer() {
	fmt.Println("Starting a new clipboard")
	listenPortString := ":" + strconv.Itoa(serverPort)
	l, err := net.Listen("tcp4", listenPortString) //nolint // complains about binding to all interfaces
	if err != nil {
		handleError(err)
		return
	}
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	fmt.Println("Run", "`uniclip -target", getOutboundIP().String()+":"+port+"`", "to join this clipboard")
	fmt.Println()
	for {
		c, err := l.Accept()
		if err != nil {
			handleError(err)
			return
		}
		fmt.Println("Connected to device at " + c.RemoteAddr().String())
		go HandleClient(c)
	}
}

// Handle a client as a server
func HandleClient(c net.Conn) {
	w := bufio.NewWriter(c)
	listOfClients = append(listOfClients, w)
	defer c.Close()
	go MonitorSentClips(bufio.NewReader(c))
	MonitorLocalClip(w)
}

// Connect to the server (which starts a new clipboard)
func ConnectToServer(address string) {
	c, err := net.Dial("tcp4", address)
	if c == nil {
		handleError(err)
		fmt.Println("Could not connect to", address)
		return
	}
	if err != nil {
		handleError(err)
		return
	}
	defer func() { _ = c.Close() }()
	fmt.Println("Connected to the clipboard")
	go MonitorSentClips(bufio.NewReader(c))
	MonitorLocalClip(bufio.NewWriter(c))
}

// monitors for changes to the local clipboard and writes them to w
func MonitorLocalClip(w *bufio.Writer) {
	for {
		localClipboard = getLocalClip()
		// logger.Debug("clipboard changed so sending it. localClipboard =", localClipboard)
		err := sendClipboard(w, localClipboard)
		if err != nil {
			handleError(err)
			return
		}
		for localClipboard == getLocalClip() {
			time.Sleep(time.Second * time.Duration(secondsBetweenChecksForClipChange))
		}
	}
}

// monitors for clipboards sent through r
func MonitorSentClips(r *bufio.Reader) {
	var foreignClipboard string
	var foreignClipboardBytes []byte
	for {
		err := gob.NewDecoder(r).Decode(&foreignClipboardBytes)
		if err != nil {
			if err == io.EOF {
				return // no need to monitor: disconnected
			}
			handleError(err)
			continue // continue getting next message
		}

		// decrypt if needed
		if encryption {
			foreignClipboardBytes, err = decrypt(password, foreignClipboardBytes)
			if err != nil {
				handleError(err)
				continue
			}
		}

		foreignClipboard = string(foreignClipboardBytes)
		// hacky way to prevent empty clipboard TODO: find out why empty cb happens
		if foreignClipboard == "" {
			continue
		}
		// foreignClipboard = decompress(foreignClipboardBytes)
		setLocalClip(foreignClipboard)
		localClipboard = foreignClipboard
		logger.Debug("rcvd:", "foreignClipboard", foreignClipboard)
		for i := range listOfClients {
			if listOfClients[i] != nil {
				err = sendClipboard(listOfClients[i], foreignClipboard)
				if err != nil {
					listOfClients[i] = nil
					fmt.Println("Error when trying to send the clipboard to a device. Will not contact that device again.")
				}
			}
		}
	}
}

// sendClipboard compresses and then if secure is enabled, encrypts data
func sendClipboard(w *bufio.Writer, clipboard string) error {
	var clipboardBytes []byte
	var err error
	clipboardBytes = []byte(clipboard)
	// clipboardBytes = compress(clipboard)
	// fmt.Printf("cmpr: %x\ndcmp: %x\nstr: %s\n\ncmpr better by %d\n", clipboardBytes, []byte(clipboard), clipboard, len(clipboardBytes)-len(clipboard))
	if encryption {
		clipboardBytes, err = encrypt(password, clipboardBytes)
		if err != nil {
			return err
		}
	}

	err = gob.NewEncoder(w).Encode(clipboardBytes)
	if err != nil {
		return err
	}
	logger.Debug("sent:", "clipboard", clipboard)
	// if secure {
	// 	logger.Debug("--secure is enabled, so actually sent as:", hex.EncodeToString(clipboardBytes))
	// }
	return w.Flush()
}

// Thanks to https://bruinsslot.jp/post/golang-crypto/ for crypto logic
func encrypt(key, data []byte) ([]byte, error) {
	key, salt, err := deriveKey(key, nil)
	if err != nil {
		return nil, err
	}
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	ciphertext = append(ciphertext, salt...)
	return ciphertext, nil
}

func decrypt(key, data []byte) ([]byte, error) {
	salt, data := data[len(data)-32:], data[:len(data)-32]
	key, _, err := deriveKey(key, salt)
	if err != nil {
		return nil, err
	}
	blockCipher, err := aes.NewCipher(key)
	if err := checkError(err); err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err := checkError(err); err != nil {
		return nil, err
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func deriveKey(password, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, err
		}
	}
	key, err := scrypt.Key(password, salt, cryptoStrength, 8, 1, 32)
	if err != nil {
		return nil, nil, err
	}
	return key, salt, nil
}

// func compress(str string) []byte {
// 	var buf bytes.Buffer
// 	zw, _ := flate.NewWriter(&buf, -1)
// 	_, _ = zw.Write([]byte(str))
// 	_ = zw.Close()
// 	return buf.Bytes()
// }

// func decompress(b []byte) string {
// 	var buf bytes.Buffer
// 	_, _ = buf.Write(b)
// 	zr := flate.NewReader(&buf)
// 	decompressed, err := ioutil.ReadAll(zr)
// 	if err != nil {
// 		handleError(err)
// 		return "Issues while decompressing clipboard"
// 	}
// 	_ = zr.Close()
// 	return string(decompressed)
// }

func runGetClipCommand() string {
	var out []byte
	var err error
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "windows": //nolint // complains about literal string "windows" being used multiple times
		cmd = exec.Command("powershell.exe", "-command", "Get-Clipboard")
	default:
		if _, err = exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-out", "-selection", "clipboard")
		} else if _, err = exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--output", "--clipboard")
		} else if _, err = exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command("wl-paste", "--no-newline")
		} else if _, err = exec.LookPath("termux-clipboard-get"); err == nil {
			cmd = exec.Command("termux-clipboard-get")
		} else {
			handleError(errors.New("sorry, uniclip won't work if you don't have xsel, xclip, wayland or Termux installed :(\nyou can create an issue at https://github.com/quackduck/uniclip/issues"))
			os.Exit(2)
		}
	}
	if out, err = cmd.Output(); err != nil {
		handleError(err)
		return "An error occurred wile getting the local clipboard"
	}
	if runtime.GOOS == "windows" {
		return strings.TrimSuffix(string(out), "\r\n") // powershell's get-clipboard adds a windows newline to the end for some reason
	}
	return string(out)
}

func getLocalClip() string {
	str := runGetClipCommand()
	// for ; str == ""; str = runGetClipCommand() { // wait until it's not empty
	// 	time.Sleep(time.Millisecond * 100)
	// }
	return str
}

func setLocalClip(s string) {
	var copyCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		copyCmd = exec.Command("pbcopy")
	case "windows":
		copyCmd = exec.Command("clip")
	default:
		if _, err := exec.LookPath("xclip"); err == nil {
			copyCmd = exec.Command("xclip", "-in", "-selection", "clipboard")
		} else if _, err = exec.LookPath("xsel"); err == nil {
			copyCmd = exec.Command("xsel", "--input", "--clipboard")
		} else if _, err = exec.LookPath("wl-copy"); err == nil {
			copyCmd = exec.Command("wl-copy")
		} else if _, err = exec.LookPath("termux-clipboard-set"); err == nil {
			copyCmd = exec.Command("termux-clipboard-set")
		} else {
			handleError(errors.New("sorry, uniclip won't work if you don't have xsel, xclip, wayland or Termux:API installed :(\nyou can create an issue at https://github.com/quackduck/uniclip/issues"))
			os.Exit(2)
		}
	}
	in, err := copyCmd.StdinPipe()
	if err != nil {
		handleError(err)
		return
	}
	if err = copyCmd.Start(); err != nil {
		handleError(err)
		return
	}
	if _, err = in.Write([]byte(s)); err != nil {
		handleError(err)
		return
	}
	if err = in.Close(); err != nil {
		handleError(err)
		return
	}
	if err = copyCmd.Wait(); err != nil {
		handleError(err)
		return
	}
}

func getOutboundIP() net.IP {
	// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go/37382208#37382208
	conn, err := net.Dial("udp", "8.8.8.8:80") // address can be anything. Doesn't even have to exist
	if err != nil {
		handleError(err)
		return nil
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func handleError(err error) {
	if err == io.EOF {
		fmt.Println("Disconnected")
	} else {
		fmt.Fprintln(os.Stderr, "error: ["+err.Error()+"]")
	}
}

// func argsHaveOption(long string, short string) (hasOption bool, foundAt int) {
// 	for i, arg := range os.Args {
// 		if arg == "--"+long || arg == "-"+short {
// 			return true, i
// 		}
// 	}
// 	return false, 0
// }

// keep order
// func removeElemFromSlice(slice []string, i int) []string {
// 	return append(slice[:i], slice[i+1:]...)
// }

func checkError(err error) error {
	if err != nil {
		return err
	}
	return nil
}
