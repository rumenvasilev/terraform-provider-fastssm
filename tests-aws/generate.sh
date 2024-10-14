#!/bin/sh

TEMPLATE=$(cat << EndOfMessage
import {
  to = aws_ssm_parameter.test[NUMBER]
  id = "/pencho/vladigerov/NUMBER"
}

EndOfMessage

)

# Function to replace the placeholder with an argument
render_template() {
  local replacement_value="$1"
  
  modified_var="${TEMPLATE//NUMBER/$replacement_value}"
  
  # Print the modified variable
  echo "$modified_var"
}

i=0
while [ $i -ne 1234 ]
do
        i=$(($i+1))
        render_template $i >> imports.tf
        echo "$i"
done
