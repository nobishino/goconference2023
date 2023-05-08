go build .
while true; do
  ./messagepassing
  exit_code=$?
  
  if [ $exit_code -ne 0 ]; then
    break
  fi
  
done