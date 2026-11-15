// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.security;

import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Passwords of the two demo accounts ({@code admin} may delete, {@code user} may not).
 * They are read from configuration, so they can be set per environment instead of being compiled in.
 */
@ConfigurationProperties("vetpet.security")
public record VetpetSecurityProperties(String adminPassword, String userPassword) {
}
