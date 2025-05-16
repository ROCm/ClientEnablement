# set minimum cmake version
cmake_minimum_required(VERSION 3.27)

# project name
project(test-rvs    LANGUAGES NONE)

# define tests
enable_testing()
include(CTest)

execute_process(
  COMMAND         hostname -f
  OUTPUT_VARIABLE SITENAME
)

# long tests:
add_test(NAME babel          COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/babel.conf )
set_tests_properties(babel  PROPERTIES LABELS "Basic;Standard") # 10 minutes

add_test(NAME gst_selfcheck  COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/gst_selfcheck.conf )
set_tests_properties(gst_selfcheck  PROPERTIES LABELS "Check") # 50 minutes

add_test(NAME gst_stress     COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/gst_stress.conf )
set_tests_properties(gst_stress  PROPERTIES LABELS "Stress") # 10 minutes

add_test(NAME iet_stress     COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/iet_stress.conf )
set_tests_properties(iet_stress  PROPERTIES LABELS "Stress") # 10 minutes

add_test(NAME gst_single     COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/gst_single.conf )
set_tests_properties(gst_single PROPERTIES LABELS "Stress") # 50 minutes

# standard tests
add_test(NAME pebb_single    COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/pebb_single.conf )
set_tests_properties(pebb_single  PROPERTIES LABELS "Standard") # 10 minutes

add_test(NAME gst_ext        COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/gst_ext.conf )
set_tests_properties(gst_ext PROPERTIES LABELS "Standard") # 2 minutes

add_test(NAME iet_single     COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/iet_single.conf )
set_tests_properties(iet_single PROPERTIES LABELS "Standard") # 5 minutes

add_test(NAME pbqt_single    COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/MI300X/pbqt_single.conf )
set_tests_properties(pbqt_single PROPERTIES LABELS "Standard") # 8 minutes

# optional tests
add_test(NAME rcqt_single    COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/rcqt_single.conf )
set_tests_properties(rcqt_single PROPERTIES LABELS "QT") #  minutes

add_test(NAME peqt_single    COMMAND  /opt/rocm/bin/rvs -c /opt/rocm/share/rocm-validation-suite/conf/peqt_single.conf )
set_tests_properties(peqt_single PROPERTIES LABELS "QT") #  minutes
      
