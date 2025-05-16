# set minimum cmake version
cmake_minimum_required(VERSION 3.27)

# project name
project(test-agfhc LANGUAGES NONE)

# define tests
enable_testing()
include(CTest)

execute_process(
  COMMAND         hostname -f
  OUTPUT_VARIABLE SITENAME
)

# turn off NUMA as that interferes some AGFHC tests
add_test(NAME disable_numa              COMMAND  sudo /usr/bin/numad -i0  )

add_test(NAME agfhc_single_pass         COMMAND  sudo /opt/amd/agfhc/agfhc -r /opt/amd/agfhc/recipes/mi300x/single_pass.yml  )
set_tests_properties(agfhc_single_pass  PROPERTIES LABELS "MI300X_single_pass")
set_tests_properties(agfhc_single_pass  PROPERTIES TIMEOUT 0)

add_test(NAME all_lvl1                  COMMAND  sudo /opt/amd/agfhc/agfhc -r /opt/amd/agfhc/recipes/mi300x/all_lvl1.yml  )
set_tests_properties(all_lvl1           PROPERTIES LABELS "MI300X_all_level1")
set_tests_properties(all_lvl1           PROPERTIES TIMEOUT 0)

add_test(NAME all_burnin_4h             COMMAND  sudo /opt/amd/agfhc/agfhc -r /opt/amd/agfhc/recipes/mi300x/all_burnin_4h.yml  )
set_tests_properties(all_burnin_4h      PROPERTIES LABELS "MI300X_all_burnin_4h")
set_tests_properties(all_burnin_4h      PROPERTIES TIMEOUT 0)

add_test(NAME enable_numa               COMMAND  sudo /usr/bin/numad -i15  )
