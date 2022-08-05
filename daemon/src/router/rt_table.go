package router

import (
	"os"
	"io/ioutil"
	"github.com/vishvananda/netlink"
	"log"
	"bufio"
	"strings"
	"sort"
	"strconv"
	"net"
	"errors"
	"fmt"
)

const (
	MIX_TABLE_INDEX = 100
	DEFAULT_RT_TABLE_PATH = "/etc/iproute2/rt_tables"
)

var RT_TABLE_PATH string = DEFAULT_RT_TABLE_PATH

func SetRTTablePath() {
	setTablePath, found := os.LookupEnv("RT_TABLE_PATH")
	if found && setTablePath != "" {
		RT_TABLE_PATH = setTablePath
	} else {
		RT_TABLE_PATH = DEFAULT_RT_TABLE_PATH
	}
}

func GetTableID(tableName string, subnet string, addIfNotExists bool) (int, error) {
	foundID, reservedIDs, err := getTableIDAndReservedIDs(tableName)
	if err != nil {
		log.Printf("failed to read %s: %v", RT_TABLE_PATH, err)
		return foundID, err
	}
	if addIfNotExists && foundID == -1 {
		foundID, err = addTable(tableName, reservedIDs)
		if err == nil {
			// delete existing rule
			deleteRule(foundID)
			err = addRule(subnet, foundID)
		} 
	}
	if foundID != -1 && !isRuleExist(foundID) {
		err = addRule(subnet, foundID)
	}
	return foundID, err
}

func DeleteTable(tableName string, tableID int) error {
	err := deleteRoutes(tableID)
	if err != nil {
		log.Printf("failed to delete routes in table %d: %v", tableID, err)
		return err
	}
	input, err := ioutil.ReadFile(RT_TABLE_PATH)
	if err != nil {
		log.Printf("failed to read %s: %v", RT_TABLE_PATH, err)
		return err
	}
	output := strings.Replace(string(input), getTableLine(tableID, tableName), "", 1)
	err = ioutil.WriteFile(RT_TABLE_PATH, []byte(output), 0666)
	if err != nil {
		log.Printf("failed to update %s: %v", RT_TABLE_PATH, err)
	}
	err = deleteRule(tableID)
	return err
}

func addRule(subnet string, tableID int) error {
	_, src, _ := net.ParseCIDR(subnet)
	rule := netlink.NewRule()
	rule.Src =  src
	rule.Table = tableID
	err := netlink.RuleAdd(rule)
	log.Printf("add rule %v:%v", rule, err)
	return err
}

func isRuleExist(tableID int) bool {
	family := netlink.FAMILY_V4
	rules, err := netlink.RuleList(family)
	if err != nil {
		return false
	}
	for _, rule := range rules {
		if rule.Table == tableID {
			return true
		}
	}
	return false
}

func getTableID(tableName string) int {
	file, err := os.Open(RT_TABLE_PATH)
	if err != nil {
		return -1
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		splited := strings.Fields(line)
		if len(splited) > 1 {
			if splited[1] == tableName {
				tableID, err := strconv.ParseInt(splited[0], 10, 64)
				if err != nil {
					return int(tableID)
				}
				return -1
			} 
		}
	}
	return -1
}

func getTableIDAndReservedIDs(tableName string) (int, []int, error) {
	foundID := -1
	reservedIDs := []int{}

	file, err := os.Open(RT_TABLE_PATH)
	if err != nil {
		return foundID, reservedIDs, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		splited := strings.Fields(line)
		if len(splited) > 1 {
			tableID, err := strconv.ParseInt(splited[0], 10, 64)
			if err != nil {
				continue
			}
			if splited[1] == tableName {
				foundID = int(tableID)
			} else {
				if tableID >= MIX_TABLE_INDEX {
					reservedIDs = append(reservedIDs, int(tableID))
				}
			}
		}
	}
	return foundID, reservedIDs, scanner.Err()
}

func addTable(tableName string, reservedIDs []int) (int, error) {
	foundID := -1
	// add table
	// 1. sort reserved ID
	sort.Ints(reservedIDs)
	// 2. find available ID
	for index, tableID := range reservedIDs {
		if index + MIX_TABLE_INDEX != tableID {
			foundID = index + MIX_TABLE_INDEX
			break
		}
	}
	if foundID != -1 {
		file, err := os.OpenFile(RT_TABLE_PATH, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			defer file.Close()
			_, err = file.WriteString(getTableLine(foundID, tableName))
		}
		return foundID, err
	}
	return foundID, errors.New("No available ID")
}

func deleteRule(tableID int) error {
	rule := netlink.NewRule()
	rule.Table = tableID
	err := netlink.RuleDel(rule)
	log.Printf("delete rule %v:%v", rule, err)
	return err
}


func deleteRoutes(tableID int) error {
	routes, err := GetRoutes(tableID)
	if err != nil {
		return err
	}
	deletedNRoute := 0
	for _, route := range routes {
		if route.Table == tableID {
			err = netlink.RouteDel(&route)
			if err == nil {
				deletedNRoute += 1
			} 
		}
	}
	log.Printf("delete %d of %d routes from table %d", deletedNRoute, len(routes), tableID)
	return nil
}


func getTableLine(tableID int, tableName string) string {
	return fmt.Sprintf("%d\t%s\n", tableID, tableName)
}