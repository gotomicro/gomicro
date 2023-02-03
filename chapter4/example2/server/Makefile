APP_NAME:=p2cserver
APP_PATH:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
SCRIPT_PATH:=$(APP_PATH)/build
COMPILE_OUT:=$(APP_PATH)/bin/$(APP_NAME)

gobuild:
	@echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>making build app<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	@chmod +x $(SCRIPT_PATH)/*.sh
	@cd $(APP_PATH) && $(SCRIPT_PATH)/gobuild.sh $(APP_NAME) $(COMPILE_OUT)
