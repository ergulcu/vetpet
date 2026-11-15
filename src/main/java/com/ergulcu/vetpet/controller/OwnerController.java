// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.controller;

import com.ergulcu.vetpet.dao.OwnerRepository;
import com.ergulcu.vetpet.dao.PetRepository;
import com.ergulcu.vetpet.model.Owner;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.validation.BindingResult;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.ModelAttribute;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.server.ResponseStatusException;
import org.springframework.web.servlet.mvc.support.RedirectAttributes;

@Controller
@RequestMapping("/owner")
public class OwnerController {

    private final OwnerRepository ownerRepository;
    private final PetRepository petRepository;

    public OwnerController(OwnerRepository ownerRepository, PetRepository petRepository) {
        this.ownerRepository = ownerRepository;
        this.petRepository = petRepository;
    }

    @GetMapping("/list")
    public String list(@RequestParam(required = false) String name, Model model) {
        model.addAttribute("name", name);
        model.addAttribute("owners", name == null || name.isBlank()
                ? ownerRepository.findAllByOrderByName()
                : ownerRepository.findByNameContainingIgnoreCaseOrderByName(name.strip()));
        return "owner/list";
    }

    @GetMapping("/create")
    public String createForm(@ModelAttribute Owner owner) {
        return "owner/create";
    }

    @PostMapping("/create")
    public String create(@Valid @ModelAttribute Owner owner, BindingResult result, RedirectAttributes redirect) {
        if (result.hasErrors()) {
            return "owner/create";
        }
        owner.setId(null);
        ownerRepository.save(owner);
        redirect.addFlashAttribute("message", "Owner " + owner.getName() + " was created");
        return "redirect:/owner/list";
    }

    @GetMapping("/update/{id}")
    public String updateForm(@PathVariable long id, Model model) {
        model.addAttribute("owner", find(id));
        return "owner/update";
    }

    @PostMapping("/update/{id}")
    public String update(@PathVariable long id, @Valid @ModelAttribute Owner owner, BindingResult result,
                         RedirectAttributes redirect) {
        find(id);
        owner.setId(id);
        if (result.hasErrors()) {
            return "owner/update";
        }
        ownerRepository.save(owner);
        redirect.addFlashAttribute("message", "Owner " + owner.getName() + " was updated");
        return "redirect:/owner/list";
    }

    @PostMapping("/delete/{id}")
    public String delete(@PathVariable long id, RedirectAttributes redirect) {
        Owner owner = find(id);
        long pets = petRepository.countByOwnerId(id);
        if (pets > 0) {
            redirect.addFlashAttribute("error",
                    owner.getName() + " still has " + pets + " pet(s); delete or move them first");
        } else {
            ownerRepository.delete(owner);
            redirect.addFlashAttribute("message", "Owner " + owner.getName() + " was deleted");
        }
        return "redirect:/owner/list";
    }

    private Owner find(long id) {
        return ownerRepository.findById(id)
                .orElseThrow(() -> new ResponseStatusException(HttpStatus.NOT_FOUND, "Owner " + id + " not found"));
    }
}
