import requests
import json
import sys
import os

checker_path_env = "CHECKER_URL"
checker_namespace_env = "CHECKER_NAMESPACE"
checker_path_fullname_env = "MULTI_NIC_HEALTH_CHECKER_ENDPOINT"
checker_timeout_fullname_env = "MULTI_NIC_HEALTH_CHECKER_TIMEOUT"

default_checker_namespace = "multi-nic-cni-operator"
default_timeout = "10" # seconds
checker_service_name = "multi-nic-cni-health-check"
service_port = 8080
service_path = "/status"

class NodeReport:
    def __init__(self, 	Info, CheckTime, Checker):
        self.info = HostCNIHealthStatus(**Info)
        self.check_time = CheckTime
        self.checker = Checker
    
    def print(self):
        self.info.print()
        if self.info.status_code == requests.codes.ok:
            print("Host is OK (all functional and connected).")
        print("Reported by {} at {}".format(self.checker,self.check_time))

class FailureReport:
    def __init__(self, HealthyHosts, FailedInfo, CheckTime, Checker):
        self.healthy_hosts = HealthyHosts
        self.failed_info = []
        for status in FailedInfo:
            info = HostCNIHealthStatus(**status)
            self.failed_info += [info]
        self.check_time = CheckTime
        self.checker = Checker

    def print(self):
        print("Found the following potential CNI failures at {}:".format(self.check_time))
        for status in self.failed_info:
            print("  ", end="")
            print_failed_status(status)
        print("Healthy hosts: ", self.healthy_hosts)

def print_failed_status(status):
    if status.allocability != len(status.connectivity):
        print("Host {} is not functional or unable to check the status: {} ({})".format(status.hostname, status.message, status.status_code))
    else:
        failedNetwork = []
        for networkAddress, connectivity in status.connectivity.items():
            if connectivity == 0:
                failedNetwork += [networkAddress]
        if len(failedNetwork) == 0:
            # wrong call
            print("Host {} seems healthy. {}".format(status.hostname, status.message)) 
        else:
            print("Host {} losts connection(s) on network address(es): {}. {}".format(status.hostname, failedNetwork, status.message))

class HostCNIHealthStatus:
    def __init__(self, HostName, Connectivity, Allocability, StatusCode, Status, Message):
        self.hostname = HostName
        self.connectivity = Connectivity
        self.allocability = Allocability
        self.status_code = StatusCode
        self.status = Status
        self.message = Message

    def print(self):
        print("===== Health Status of {} =====".format(self.hostname))
        expected_healthy_device = len(self.connectivity)
        unconnect_net_addr = [ addr for addr, val in self.connectivity.items() if not val]
        print("Allocatable network devices: {}/{}".format(self.allocability, expected_healthy_device))
        print("Connectable network devices: {}/{}".format(expected_healthy_device - len(unconnect_net_addr), expected_healthy_device))
        if len(unconnect_net_addr) > 0:
            print_failed_status(self)

def process_status_with_hostname(checker_service_path, hostname, timeout):
    response = requests.get(checker_service_path, params={'host': hostname}, timeout=timeout)
    # Check the status code
    if response.status_code == requests.codes.ok:
        # Parse the JSON response
        try:
            data = json.loads(response.text)
            report = NodeReport(**data)
            report.print()
        except Exception as err:
            print("Failed to process response: ", err)
            print(response.text)
    else:
        print("Request failed with status code:", response.status_code)
        print(response.text)

def process_status(checker_service_path, timeout):
    response = requests.get(checker_service_path, timeout=timeout)

    # Check the status code
    if response.status_code == requests.codes.ok:
        print(response.text)
    else:
        # Parse the JSON response
        try:
            data = json.loads(response.text)
            report = FailureReport(**data)
            report.print()
        except Exception as err:
            print("Failed to process failure response: ", err)
            print(response.text)
        
if __name__ == "__main__":
    checker_service_path = os.environ.get(checker_path_env, "")
    if checker_service_path == "":
        # expected to run inside the same cluster with checker
        checker_namespace = os.environ.get(checker_namespace_env, default_checker_namespace)
        checker_service_path = "http://{}.{}.svc:{}/{}".format(checker_service_name, checker_namespace, service_port, service_path)

        # may specify checker by pod ip
        # checker_pod = <pod ip>
        # checker_service_path = "http://{}.{}.pod:{}/{}".format(checker_pod, checker_namespace, service_port, service_path)

    # override if checker_path_fullname_env set
    checker_service_path = os.environ.get(checker_path_fullname_env, checker_service_path)

    if checker_service_path != "":
        try: 
            timeout = int(os.environ.get(checker_timeout_fullname_env, default_timeout))
            if len(sys.argv) == 1:
                # no hostname specified, get all status
                process_status(checker_service_path, timeout=timeout)
            else:
                hostname = sys.argv[1]
                process_status_with_hostname(checker_service_path, hostname, timeout=timeout)
        except:
            print("cannot request status from {}.".format(checker_service_path))
    else:
        print("checker_service_path is not set.")