input=addon/adapter.go
output=${1:-docs/binding.txt}

#
# Determine type aliased types.
#
types=$(awk \
	'match($0,/(type\s+)(.+)(\s+=)/,g) {print g[2]}' \
	${input} \
       | grep -v Client)

#
# Generate adapter.
#
go doc --all $(dirname ${input}) > ${output}

#
# Generate aliased types.
#
set -v
for name in ${types}
do
  echo "go doc binding.${name} >> ${output}"
  echo "go doc api.${name} >> ${output}"
done
