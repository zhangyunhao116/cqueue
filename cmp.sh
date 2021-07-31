benchok -run="sh .bench.sh > latest.txt" -maxerr=20 -ignore="DequeueOnlyEmpty" latest.txt
benchstat base.txt latest.txt
