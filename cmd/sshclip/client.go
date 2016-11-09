package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/tweekmonster/sshclip"
	"golang.org/x/crypto/ssh"
)

func getKeys(ch ssh.Channel) ([]sshclip.KeyReviewItem, error) {
	if _, err := ch.Write(sshclip.OpHeader(sshclip.OpList)); err != nil {
		return nil, err
	}

	op, err := sshclip.ReadOp(ch)
	if err != nil {
		return nil, err
	}

	var keys []sshclip.KeyReviewItem

	switch op {
	case sshclip.OpSuccess:
		dec := gob.NewDecoder(ch)
		if err := dec.Decode(&keys); err != nil {
			return nil, err
		}
	case sshclip.OpErr:
		sshclip.Dlog("Error!")
		return nil, sshclip.ReadError(ch)
	default:
		return nil, fmt.Errorf("Unknown op: %02x", op)
	}

	return keys, nil
}

func input(prompt string) (string, error) {
	stdin := bufio.NewReader(os.Stdin)
	fmt.Print("\n", prompt)
	inStr, err := stdin.ReadString('\n')
	if err != nil {
		return inStr, err
	}

	inStr = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r > 125 {
			return -1
		}
		return r
	}, inStr))

	return inStr, nil
}

func yesno(prompt string, yes bool) (bool, error) {
	if yes {
		prompt += " [Y/n]: "
	} else {
		prompt += " [y/N]: "
	}

	for {
		yn, err := input(prompt)
		if err != nil {
			return false, err
		}

		yn = strings.ToLower(yn)
		if yn[0] != 'y' && yn[0] != 'n' {
			continue
		}

		return (yes && yn[0] == 'y') || (!yes && yn[0] == 'n'), nil
	}
}

func manageKeys(host string, port int) error {
	conn, err := sshclip.SSHClientConnect(host, port)
	if err != nil {
		return err
	}

	ch, reqs, err := conn.OpenChannel("sshclip-keys", nil)
	if err != nil {
		return err
	}

	go ssh.DiscardRequests(reqs)

	var status string
	w := tabwriter.NewWriter(color.Output, 0, 4, 1, ' ', tabwriter.AlignRight)

	signer, err := sshclip.GetClientKey()
	if err != nil {
		return err
	}

	clientFingerPrint := sshclip.FingerPrintBytes(signer.PublicKey())

	printKey := func(i int, k sshclip.KeyReviewItem) {
		state := k.State
		if state == "authorized" {
			state = color.GreenString(state)
		} else if state == "rejected" {
			state = color.RedString(state)
		}

		keyString := color.YellowString(base64.StdEncoding.EncodeToString(k.FingerPrint))

		fmt.Fprintf(w, "%2d. %s (%s)", i+1, keyString, state)
		if bytes.Equal(clientFingerPrint, k.FingerPrint) {
			fmt.Fprintf(w, " (%s)", color.CyanString("you"))
		}
		fmt.Fprint(w, "\n")
		fmt.Fprintf(w, "\t\tIP:\t %s\n", k.IP)
		fmt.Fprintf(w, "\t\tAdded:\t %s\n", k.Added.Format("Jan 2, 2006 15:04 MST"))
		w.Flush()
	}

	for {
		keys, err := getKeys(ch)
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			return fmt.Errorf("No keys to manage")
		}

		fmt.Println("\nSelect a key:")

		for i, k := range keys {
			printKey(i, k)
		}

		if status != "" {
			fmt.Print("\n", status, "\n")
			status = ""
		}

		inStr, err := input("Select Key (^D or empty to quit): ")
		if err != nil {
			return err
		}

		if inStr == "" {
			return nil
		}

		i, err := strconv.Atoi(inStr)
		if err != nil {
			continue
		}

		if i < 1 || i > len(keys) {
			continue
		}

		i--
		opKey := keys[i]
		if bytes.Equal(clientFingerPrint, opKey.FingerPrint) {
			status = color.RedString("Error: Can't manage your own key")
			continue
		}

		fmt.Println("Selected Key:")
		printKey(i, opKey)

		op := sshclip.OpErr

		for {
			action, err := input("[A]ccept, [R]eject? ")
			if err != nil {
				return err
			}

			action = strings.ToLower(action)
			if action == "" {
				break
			}

			if action[0] == 'a' {
				op = sshclip.OpAccept
			} else if action[0] == 'r' {
				op = sshclip.OpReject
			} else {
				continue
			}

			break
		}

		if op == sshclip.OpErr {
			continue
		}

		payload := sshclip.OpHeader(op)
		payload = append(payload, opKey.FingerPrint...)
		if _, err := ch.Write(payload); err != nil {
			return err
		}

		res, err := sshclip.ReadOp(ch)
		if err != nil {
			return err
		}

		switch res {
		case sshclip.OpSuccess:
			var action string
			if op == sshclip.OpAccept {
				action = "authorized"
			} else {
				action = "rejected"
			}
			status = fmt.Sprintf(color.GreenString("Key #%d has been %s"), i+1, action)
		case sshclip.OpErr:
			status = fmt.Sprintf(color.RedString("Error: %s"), sshclip.ReadError(ch))
		default:
			return fmt.Errorf("Unknown op: %02x", op)
		}
	}
}
