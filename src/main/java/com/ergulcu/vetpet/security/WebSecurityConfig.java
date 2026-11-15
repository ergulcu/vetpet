// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.security;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpMethod;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.crypto.factory.PasswordEncoderFactories;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.provisioning.InMemoryUserDetailsManager;
import org.springframework.security.web.SecurityFilterChain;

/**
 * Component based configuration ({@code WebSecurityConfigurerAdapter} no longer exists).
 * Deleting is a POST protected by the CSRF token and limited to the ADMIN role.
 */
@Configuration
public class WebSecurityConfig {

    @Bean
    SecurityFilterChain securityFilterChain(HttpSecurity http) throws Exception {
        http
                .authorizeHttpRequests(auth -> auth
                        .requestMatchers("/webjars/**", "/css/**", "/login", "/error").permitAll()
                        .requestMatchers(HttpMethod.POST, "/owner/delete/**", "/pet/delete/**").hasRole("ADMIN")
                        .anyRequest().authenticated())
                .formLogin(form -> form.loginPage("/login").permitAll())
                .logout(logout -> logout.logoutSuccessUrl("/login?logout").permitAll())
                .exceptionHandling(ex -> ex.accessDeniedPage("/access-denied"));
        return http.build();
    }

    @Bean
    PasswordEncoder passwordEncoder() {
        return PasswordEncoderFactories.createDelegatingPasswordEncoder();
    }

    @Bean
    UserDetailsService userDetailsService(VetpetSecurityProperties properties, PasswordEncoder encoder) {
        return new InMemoryUserDetailsManager(
                User.withUsername("user").password(encoder.encode(properties.userPassword())).roles("USER").build(),
                User.withUsername("admin").password(encoder.encode(properties.adminPassword())).roles("ADMIN").build());
    }
}
