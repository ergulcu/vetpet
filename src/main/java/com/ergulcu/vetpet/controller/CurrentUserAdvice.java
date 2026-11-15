// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.controller;

import org.springframework.security.core.Authentication;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ModelAttribute;

/**
 * Puts the signed-in user into every model; it replaces {@code #httpServletRequest.remoteUser},
 * which Thymeleaf 3.1 no longer offers. {@code isAdmin} lets the pages hide the delete buttons.
 */
@ControllerAdvice
public class CurrentUserAdvice {

    @ModelAttribute("currentUser")
    public String currentUser(Authentication authentication) {
        return authentication == null ? null : authentication.getName();
    }

    @ModelAttribute("isAdmin")
    public boolean isAdmin(Authentication authentication) {
        return authentication != null && authentication.getAuthorities().stream()
                .map(GrantedAuthority::getAuthority)
                .anyMatch("ROLE_ADMIN"::equals);
    }
}
