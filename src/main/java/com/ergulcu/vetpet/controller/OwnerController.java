package com.ergulcu.vetpet.controller;

import com.ergulcu.vetpet.dao.OwnerRepository;
import com.ergulcu.vetpet.model.Owner;
import javax.validation.Valid;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.validation.BindingResult;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;

/**
 *
 * @author Mehmet ERGÜLCÜ
 */
@Controller
public class OwnerController {

    @Autowired
    private OwnerRepository ownerRepository;

    @GetMapping("/owner/list")
    public String listOwner(Owner owner, Model model) {
        model.addAttribute("owners", ownerRepository.findAll());
        return "owner/list";
    }
    
    @PostMapping("/owner/list")
    public String filterOwner(Owner owner, Model model) {
        model.addAttribute("owners", ownerRepository.findByNameContainingIgnoreCase(owner.getName()));
        return "owner/list";
    }

    @GetMapping("/owner/create")
    public String addOwner(Owner owner) {
        return "owner/create";

    }

    @PostMapping("/owner/create")
    public String addOwner(@Valid Owner owner, BindingResult result,Model model) {
        if (result.hasErrors()) {
            return "owner/create";
        }
        ownerRepository.save(owner);
        return "redirect:/owner/list";
    }

    @GetMapping("/owner/update/{id}")
    public String showUpdateForm(@PathVariable("id") long id, Model model) {
        Owner owner = ownerRepository.findById(id).orElseThrow(() -> new IllegalArgumentException("Invalid owner Id:" + id));
        model.addAttribute("owner", owner);
        return "owner/update";
    }

    @PostMapping("/owner/update/{id}")
    public String updateOwner(@PathVariable("id") long id, @Valid Owner owner, BindingResult result) {
        if (result.hasErrors()) {
            owner.setId(id);
            return "owner/update";
        }
        ownerRepository.save(owner);
        return "redirect:/owner/list";
    }

    @GetMapping("/owner/delete/{id}")
    public String deleteOwner(@PathVariable("id") long id) {
        Owner owner = ownerRepository.findById(id).orElseThrow(() -> new IllegalArgumentException("Invalid owner Id:" + id));
        ownerRepository.delete(owner);
        return "redirect:/owner/list";
    }
}
