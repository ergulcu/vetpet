package com.ergulcu.vetpet.controller;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;

/**
 *
 * @author Mehmet ERGÜLCÜ
 */
@Controller
public class HomeController {

    @GetMapping("/")
    public String root() {
        return "redirect:/owner/list";
    }

    @GetMapping("/user")
    public String userIndex() {
        return "user/index";
    }

    @GetMapping("/login")
    public String login() {
        return "login";
    }

    @GetMapping("/access-denied")
    public String accessDenied() {
        return "/error/access-denied";
    }

}
