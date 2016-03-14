#bin/bash

tab="--tab"
cmd="bash -c 'assignment';bash"
foo=""

for i in 1 2 4 5; do
      foo+=($tab -e "$cmd $i")         
done

gnome-terminal "${foo[@]}"

exit 0