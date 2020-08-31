FROM nixos/nix:2.3.6

# https://github.com/sgerrand/alpine-pkg-glibc
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.32-r0/glibc-2.32-r0.apk && \
    apk add glibc-2.32-r0.apk

#USER guest
WORKDIR /root

# copy all files
ADD . .
RUN nix-shell --pure -p cacert cachix curl git jq nix --run "bash install.sh"

EXPOSE 8545
ENTRYPOINT ["/startup_script_wrapper.sh"]
