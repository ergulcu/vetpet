// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

@SpringBootApplication
@ConfigurationPropertiesScan
public class VetpetApplication {

    public static void main(String[] args) {
        SpringApplication.run(VetpetApplication.class, args);
    }
}
