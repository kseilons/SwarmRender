DIRS = keycloak

all: $(DIRS)

$(DIRS):
	"$(MAKE)" -C $@

clean:
	"$(MAKE)" -C keycloak clean

.PHONY: all clean $(DIRS)