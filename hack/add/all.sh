
host="localhost:8080"

#######################################################
# ALL
#######################################################

dir=`dirname $0`
cd ${dir}

./job-function.sh
./stakeholder-group.sh
./stakeholder.sh
./business-service.sh
./application.sh
./bucket.sh
./review.sh
./proxy.sh
