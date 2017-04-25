#!/bin/sh
if [ -e ~/.bashrc ];then
    source ~/.bashrc
fi
Exec=mlt-server
ip="10.134.11.83"
port=18080
log_dir=log

exec_prefix=${Exec%%.*}
logdir=../log
TM=$(date +'%F %T')
machine=$(uname -n)
if ! pgrep -f "(^|/)${Exec}($| ).*${ip}.*{$port}" > /dev/null;then
    nohup ./${Exec} -addr=$ip -debug=1 -port=$port 2>&1 | ./cronolog -p 30days -l "$log_dir/${exec_prefix}.log" "$log_dir/${exec_prefix}.%Y-%m-%d" 2>>err.log &
    echo $!
    sleep 0.5
    if pgrep -f "(^|/)${Exec}($| ).*${ip}.*{$port}" > /dev/null;then
        info="$TM $Exec start succ"
        echo $info
    else
        info="$TM $Exec start failed"
        echo $info
    fi
else
    echo "$TM $Exec already started!"
fi
