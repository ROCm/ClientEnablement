import argparse
import re
import os
import subprocess
import socket

"""
Parse output from jobs that execute under CTest and collect pass/fail info.
Data is passed to Prometheus via a push gateway defined by the user.

example: python3 testResultsParser.py  --pushgw "http://10.200.116.42:9091" --datadir ./OUTPUT

"""


class TestResultParser:
    """A flexible parser for test result files that extracts key information using regex patterns."""

    def __init__(self, gw):
        # Define regex patterns for common test result formats
        self.patterns = {
            # Basic Attribute patterns
            "site": r'Site:\s*(.*)',
            "build_name" : r'Build name:\s*(.*)',

            # Test Results
            "test-summary":r"(\d{1,3})% tests passed, (\d+) tests failed out of (\d+)",

        }
        self.PUSHGATEWAY_URL = gw
        self.JOB_NAME = "test-attributes"
        self.INSTANCE = f"{socket.gethostname()}"
        #print(f"Debug: In init gw is {gw} ")

    def parse_file(self, file_path):
        """Parse a test result file and extract relevant information."""
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"Error: Test result file not found: {file_path}")

        # Read the content of the file
        with open(file_path, 'r', encoding='utf-8', errors='replace') as file:
            content = file.read()

        # Extract information
        result_info = {
        "site": None,
        "test-suite": None,
        "summary": None,
        "test_results": {}
            }

        # Split the output into lines
        lines = content.strip().split('\n')

        # Process each line
        for i in range(len(lines)):
            line = lines[i].strip()

            # Extract Site information
            site_match = re.search(self.patterns["site"], line)
            if site_match:
                result_info["site"] = site_match.group(1).strip()


            # Extract Build Name
            build_match = re.search(self.patterns["build_name"], line)
            if build_match:
                result_info["test-suite"] = build_match.group(1).strip()

            # Extract summary result (usually at the end)
            summary_match = re.search(self.patterns["test-summary"], line, re.IGNORECASE)
            if summary_match:
                result_info["summary"] = {
                    "percentage-passed": summary_match.group(1),
                    "failed_count": summary_match.group(2),
                    "total_count": summary_match.group(3),
                }

            # Pattern for test line: digits/digits Test #digits: test_name ...
            test_match = re.match(r'\d+/\d+\s+Test\s+#\d+:\s+(\S+.*?)(?:\s*\.+)', line)

            if test_match and i < len(lines) - 1:
                test_name = test_match.group(1).strip()
                result_line = lines[i + 1].strip()

                # Check result
                if "Passed" in result_line:
                    result_info["test_results"][test_name] = "pass"
                else:  # Failed or Not Run or any other case
                    result_info["test_results"][test_name] = "fail"



        return result_info


    # generate promethus format metrics

    def generate_prometheus_metrics(self, attribute_dictionary:dict):

        if not attribute_dictionary or attribute_dictionary['summary'] == None :
            print("Warning: !!!!!No attribute data found!!!!!!!")
            return


        metrics = []
        site = attribute_dictionary['site']
        test_suite = attribute_dictionary['test-suite']
        summary = attribute_dictionary['summary']
        test_results = attribute_dictionary["test_results"]



        # Summary for the test suite (pass, fail, total)
        metrics.append(f'test_suite_summary_attributes{{site="{site}", test_suite="{test_suite}", result="pass_percentage"}} {summary["percentage-passed"]}')
        metrics.append(f'test_suite_summary_attributes{{site="{site}", test_suite="{test_suite}", result="fail_count"}} {summary["failed_count"]}')
        metrics.append(f'test_suite_summary_attributes{{site="{site}", test_suite="{test_suite}", result="total_count"}} {summary["total_count"]}')


        #  appending individual test results
        for key, value in test_results.items():
            if value == 'pass':
                result_gauge_value = 1
            else:
                result_gauge_value = 0
            metrics.append(f'test_suite_summary_attributes{{site="{site}", test_suite="{test_suite}", test_name="{key}"}} {result_gauge_value}')

        # Return all generated metrics as a string (joined by newline for readability)
        return "\n".join(metrics)  + "\n"


    def push_to_gateway(self, metrics_data):
        if not metrics_data:
            print("Warning: No metrics to push.")
            return


        # payload = "\n".join(metrics_lines) + "\n"
        payload = metrics_data
        url = f"{self.PUSHGATEWAY_URL}/metrics/job/{self.JOB_NAME}/instance/{self.INSTANCE}"

        print(f"Info: Pushing all metrics for node {self.INSTANCE}...")
        status = subprocess.run(['curl', '--data-binary', '@-', url], input=payload.encode())

        return status





def main():

    parser = argparse.ArgumentParser(description="Parse GPU Performance CTest output to Prometheus.")
    parser.add_argument('--datadir', type=str, help='The name of directory containing the data to be parsed.')
    parser.add_argument('--pushgw',  type=str, help='The full URL of the Prometheus push gateway.')

    args = parser.parse_args()

    try:
        test_parser = TestResultParser(gw=args.pushgw)
        for file in os.listdir(args.datadir):
            filename = file
            full_path = os.path.join(args.datadir, file)
            if os.path.isfile(full_path):
                results = test_parser.parse_file(full_path)

            print(f"\nInfo: Parsed Test Results for {filename}")
            print("="*50)

            print(results)
            print("Debug: generate formatted metrics")
            formatted_metrics = test_parser.generate_prometheus_metrics( results)
            print("="*50)
            print(formatted_metrics)
            print(test_parser.push_to_gateway(formatted_metrics))


    except Exception as e:
        print(f"Error: {e}")
        return 1

    return 0


if __name__ == "__main__":
    exit(main())

