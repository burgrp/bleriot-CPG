ifndef BLERIOT_MAKE

$(or $(MAKECMDGOALS),build): __bleriot_bounce
	@:
.PHONY: __bleriot_bounce $(MAKECMDGOALS)
__bleriot_bounce:
	@go run . make $(MAKECMDGOALS)

else

IMAGE = image.elf
PYOCD_FREQUENCY ?= 100k

.PHONY: build flash gdb rtt install-pack disassembly new hub

build:
	tinygo build --size html --serial rtt --scheduler none -print-allocs=. --gc leaking --target ${TARGET_TINYGO} -o ${IMAGE} .

flash: build
	pyocd load -t ${TARGET_PYOCD} -f ${PYOCD_FREQUENCY} -M under-reset ${IMAGE}
	pyocd rtt -t ${TARGET_PYOCD} -f ${PYOCD_FREQUENCY} -M attach

gdb:
	pyocd gdb --target ${TARGET_PYOCD} -f ${PYOCD_FREQUENCY} -M attach --semihosting --persist

rtt:
	pyocd rtt -t ${TARGET_PYOCD} -f ${PYOCD_FREQUENCY} -M attach

install-pack:
	pyocd pack install ${CMSIS_PACK}

disassembly: build
	arm-none-eabi-objdump -dS ${IMAGE} > disassembly.txt

new:
	go run . new

hub:
	go run . hub --diagnostics rf

endif