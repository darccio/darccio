#!/bin/bash

hugo
cd public/
minify -r -v -a -o . .
aws s3 sync . s3://dario.im
cd -
