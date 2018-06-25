#!/bin/sh
### BEGIN INIT INFO
# Provides:          labean
# Required-Start:    $network $syslog $local_fs $time
# Required-Stop:     $network $syslog $local_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Description:       <DESCRIPTION>
### END INIT INFO

PIDFILE=/var/run/labean.pid
CMD=/usr/sbin/labean
CONFIG=/etc/labean.conf

start() {
  if [ -f /var/run/$PIDNAME ] && kill -0 $(cat /var/run/$PIDNAME); then
    echo 'Labean already running' >&2
    return 1
  fi
  echo 'Starting Labean…' >&2
  nohup "$CMD" "$CONFIG" > /dev/null &
  echo "$!" > "$PIDFILE"
}

stop() {
  if [ ! -f "$PIDFILE" ] || ! kill -0 $(cat "$PIDFILE"); then
    echo 'Labean not running' >&2
    return 1
  fi
  echo 'Stopping Labean…' >&2
  kill -15 $(cat "$PIDFILE") && rm -f "$PIDFILE"
  echo 'Labean stopped' >&2
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    stop
    start
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
esac
