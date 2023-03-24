#!/bin/bash

read -p "email: " email
read -s -p "Password: " pass; echo

pass_base64=$(echo "${email}:${pass}"|base64)
header="Authorization: Basic ${pass_base64}"
echo $pass_base64
echo $header

curl --location --request POST 'https://api.azionapi.net/tokens' --header 'Accept: application/json; version=3' --header "$header"
